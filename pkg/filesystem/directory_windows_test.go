package filesystem

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// TestDirectoryLongPaths tests a variety of Directory operations on directory
// and file names that exceed the default Windows path length limit.
func TestDirectoryLongPaths(t *testing.T) {
	// Create a temporary directory (that will be automatically removed).
	temporaryDirectoryPath := t.TempDir()

	// Create a directory in the temporary directory with a name that will
	// exceed the Windows path length limit.
	longDirectoryName := strings.Repeat("d", windowsLongPathTestingLength)
	if err := os.Mkdir(filepath.Join(temporaryDirectoryPath, longDirectoryName), 0700); err != nil {
		t.Fatal("unable to create test directory with long name:", err)
	}

	// Create a file in the temporary directory with a name that will exceed the
	// Windows path length limit.
	longFileName := strings.Repeat("f", windowsLongPathTestingLength)
	file, err := os.Create(filepath.Join(temporaryDirectoryPath, longFileName))
	if err != nil {
		t.Fatal("unable to create test file with long name:", err)
	}
	file.Close()

	// Open the temporary directory for access and defer its closure.
	closer, _, err := Open(temporaryDirectoryPath, false)
	if err != nil {
		t.Fatal("unable to open directory:", err)
	}
	defer closer.Close()

	// Extract the directory object.
	var directory *Directory
	if d, ok := closer.(*Directory); !ok {
		t.Fatal("opened object is not a directory")
	} else {
		directory = d
	}

	// Access the internal directory and ensure that doing so succeeds.
	if d, err := directory.OpenDirectory(longDirectoryName); err != nil {
		t.Error("unable to open directory with long name:", err)
	} else {
		d.Close()
	}

	// Access the internal file and ensure that doing so succeeds.
	if f, _, err := directory.OpenFile(longFileName); err != nil {
		t.Error("unable to open file with long name:", err)
	} else {
		f.Close()
	}

	// Try to set permissions.
	self, err := user.Current()
	if err != nil {
		t.Fatal("unable to access current user:", err)
	}
	ownership, err := NewOwnershipSpecification("sid:"+self.Uid, "")
	if err != nil {
		t.Fatal("unable to construct ownership specification")
	}
	if err := directory.SetPermissions(longFileName, ownership, 0660); err != nil {
		t.Error("unable to set permissions for file with long name:", err)
	}

	// Ensure that removing the internal directory succeeds.
	if err := directory.RemoveDirectory(longDirectoryName); err != nil {
		t.Error("unable to remove directory with long name:", err)
	}

	// Ensure that removing the internal file succeeds.
	if err := directory.RemoveFile(longFileName); err != nil {
		t.Error("unable to remove file with long name:", err)
	}
}

// TestDirectoryRemoveFileReadOnly verifies that a file with the read-only
// attribute set can be removed. Git creates loose object files with mode 0444,
// which Git for Windows stores as the read-only attribute, so this is a very
// common situation in practice and failing to handle it makes such files
// impossible to delete (and, during synchronization, produces a transition
// problem that is retried and fails on every cycle).
func TestDirectoryRemoveFileReadOnly(t *testing.T) {
	// Create a temporary directory (that will be automatically removed).
	temporaryDirectoryPath := t.TempDir()

	// Create a test file.
	const name = "readonly"
	path := filepath.Join(temporaryDirectoryPath, name)
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatal("unable to create test file:", err)
	}

	// Convert the path to UTF-16 for the attribute calls.
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal("unable to convert path to UTF-16:", err)
	}

	// Set the read-only attribute on the file and restore the original
	// attributes when the test completes.
	attributes, err := windows.GetFileAttributes(path16)
	if err != nil {
		t.Fatal("unable to read file attributes:", err)
	}
	defer windows.SetFileAttributes(path16, attributes)
	if err := windows.SetFileAttributes(path16, attributes|windows.FILE_ATTRIBUTE_READONLY); err != nil {
		t.Fatal("unable to set read-only attribute:", err)
	}

	// Open the temporary directory for access and defer its closure.
	closer, _, err := Open(temporaryDirectoryPath, false)
	if err != nil {
		t.Fatal("unable to open directory:", err)
	}
	defer closer.Close()

	// Extract the directory object.
	directory, ok := closer.(*Directory)
	if !ok {
		t.Fatal("opened object is not a directory")
	}

	// Ensure that removal succeeds.
	if err := directory.RemoveFile(name); err != nil {
		t.Fatal("unable to remove read-only file:", err)
	}

	// Ensure that the file is gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file still exists after removal")
	}
}
