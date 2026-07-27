//go:build windows

package loadstrike

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func openRuntimeObjectNoFollow(path string, directory bool) (*os.File, error) {
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	flags := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	handle, err := windows.CreateFile(
		pathPointer,
		windows.GENERIC_READ|windows.READ_CONTROL|windows.WRITE_DAC,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		flags,
		0,
	)
	if err != nil {
		return nil, err
	}
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &information); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, err
	}
	if information.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		_ = windows.CloseHandle(handle)
		return nil, fmt.Errorf("runtime cache path is a reparse point: %s", path)
	}
	return os.NewFile(uintptr(handle), path), nil
}

func runtimeObjectIdentityForFile(file *os.File) (runtimeObjectIdentity, error) {
	var information windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(
		windows.Handle(file.Fd()),
		&information,
	); err != nil {
		return runtimeObjectIdentity{}, err
	}
	return runtimeObjectIdentity{
		device: uint64(information.VolumeSerialNumber),
		file: uint64(information.FileIndexHigh)<<32 |
			uint64(information.FileIndexLow),
		links: uint64(information.NumberOfLinks),
	}, nil
}

func validateRuntimeObjectSecurity(
	file *os.File,
	directory bool,
	requirePrivateAccess bool,
	requireProtectedAccess bool,
) error {
	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION,
	)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	currentUser, err := currentWindowsUserSID()
	if err != nil {
		return err
	}
	if owner == nil || !owner.Equals(currentUser) {
		return fmt.Errorf("runtime cache object is not owned by the current user: %s", file.Name())
	}
	control, _, err := descriptor.Control()
	if err != nil {
		return err
	}
	if requireProtectedAccess &&
		control&windows.SE_DACL_PROTECTED == 0 {
		return fmt.Errorf(
			"runtime cache object DACL is not protected: %s",
			file.Name(),
		)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	if dacl == nil {
		return fmt.Errorf("runtime cache object has an unrestricted DACL: %s", file.Name())
	}
	var approvedWriters []*windows.SID
	if requirePrivateAccess {
		approvedWriters, err = approvedWindowsWriterSIDs(currentUser)
		if err != nil {
			return err
		}
	}
	const fileDeleteChild = uint32(0x00000040)
	const writeMask = uint32(windows.GENERIC_WRITE|
		windows.GENERIC_ALL|
		windows.FILE_WRITE_DATA|
		windows.FILE_APPEND_DATA|
		windows.FILE_WRITE_ATTRIBUTES|
		windows.FILE_WRITE_EA|
		windows.WRITE_DAC|
		windows.WRITE_OWNER|
		windows.DELETE) | fileDeleteChild
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &ace); err != nil {
			return err
		}
		if ace == nil {
			return fmt.Errorf("runtime cache object contains an invalid DACL entry: %s", file.Name())
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			if requirePrivateAccess || requireProtectedAccess {
				return fmt.Errorf(
					"runtime cache object contains an unsupported DACL entry: %s",
					file.Name(),
				)
			}
			continue
		}
		if !requirePrivateAccess || uint32(ace.Mask)&writeMask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		approved := false
		for _, writer := range approvedWriters {
			if sid.Equals(writer) {
				approved = true
				break
			}
		}
		if !approved {
			return fmt.Errorf(
				"runtime cache object grants write or delete-child access to an unapproved Windows principal: %s",
				file.Name(),
			)
		}
	}
	return nil
}

func protectRuntimeObject(path string, directory bool, _ os.FileMode) error {
	descriptor, err := privateRuntimeSecurityDescriptor(directory)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

func protectOpenedRuntimeObject(
	file *os.File,
	directory bool,
	_ os.FileMode,
) error {
	descriptor, err := privateRuntimeSecurityDescriptor(directory)
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	)
}

func createProtectedRuntimeDirectory(path string) error {
	descriptor, err := privateRuntimeSecurityDescriptor(true)
	if err != nil {
		return err
	}
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	return windows.CreateDirectory(pathPointer, &attributes)
}

func privateRuntimeSecurityDescriptor(
	directory bool,
) (*windows.SECURITY_DESCRIPTOR, error) {
	currentUser, err := currentWindowsUserSID()
	if err != nil {
		return nil, err
	}
	userRights := "GRGXSD"
	inheritance := ""
	if directory {
		userRights = "FA"
		inheritance = "OICI"
	}
	sddl := fmt.Sprintf(
		"D:P(A;%s;FA;;;SY)(A;%s;FA;;;BA)(A;%s;%s;;;%s)",
		inheritance,
		inheritance,
		inheritance,
		userRights,
		currentUser.String(),
	)
	descriptor, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		return nil, err
	}
	return descriptor, nil
}

func currentWindowsUserSID() (*windows.SID, error) {
	var token windows.Token
	if err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_QUERY,
		&token,
	); err != nil {
		return nil, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return user.User.Sid.Copy()
}

func approvedWindowsWriterSIDs(currentUser *windows.SID) ([]*windows.SID, error) {
	types := []windows.WELL_KNOWN_SID_TYPE{
		windows.WinLocalSystemSid,
		windows.WinBuiltinAdministratorsSid,
	}
	principals := make([]*windows.SID, 0, len(types)+1)
	principals = append(principals, currentUser)
	for _, sidType := range types {
		sid, err := windows.CreateWellKnownSid(sidType)
		if err != nil {
			return nil, err
		}
		principals = append(principals, sid)
	}
	return principals, nil
}
