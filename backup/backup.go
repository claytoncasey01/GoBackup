package backup

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
)

// Backup is used to represent the current backup configuration
type Backup struct {
	SourcePath         string `json:"sourcePath"`
	BackupPath         string `json:"backupPath"`
	TimeBetweenBackups int    `json:"timeBetweenBackups"`
}

// NewBackup creates an empty Backup struct pointer
func NewBackup() *Backup {
	return &Backup{}
}

// CheckChanged walks the directory to be backed up and figure out what
// files have been hanged and store those off.
func (b *Backup) CheckChanged() {
	// Walk the source directory and create the backups.
	filepath.Walk(b.SourcePath, walkSource(b))
}

// FileChecksum Takes the sourceFile and the backedUp version of the file and
//  generates an md5 hash. Those are then compared to see if they are equal.
func (b *Backup) FileChecksum(sourceFilePath string, backupFilePath string) bool {
	filechunk := 8192

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		log.Fatal(err)
	}
	backupFile, err := os.Open(backupFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFile.Close()
	defer backupFile.Close()

	// calculate the file size
	sourceFileInfo, _ := sourceFile.Stat()
	backupFileInfo, _ := backupFile.Stat()

	sourceFileSize := sourceFileInfo.Size()
	backupFileSize := backupFileInfo.Size()

	sourceFileBlocks := uint64(math.Ceil(float64(sourceFileSize) / float64(filechunk)))
	backupFileBlocks := uint64(math.Ceil(float64(backupFileSize) / float64(filechunk)))

	sourceFileHash := md5.New()
	backupFileHash := md5.New()

	// Get the MD5 hash for the current file
	for i := uint64(0); i < sourceFileBlocks; i++ {
		blocksize := int(math.Min(float64(filechunk), float64(sourceFileSize-int64(i*uint64(filechunk)))))
		buf := make([]byte, blocksize)
		sourceFile.Read(buf)
		io.WriteString(sourceFileHash, string(buf)) // append into hash
	}
	// Get the MD5 hash for the backed up file
	for i := uint64(0); i < backupFileBlocks; i++ {
		blocksize := int(math.Min(float64(filechunk), float64(backupFileSize-int64(i*uint64(filechunk)))))
		buf := make([]byte, blocksize)
		backupFile.Read(buf)
		io.WriteString(backupFileHash, string(buf)) // append into hash
	}

	sourceFileDigest := fmt.Sprintf("%x", sourceFileHash.Sum(nil))
	backupDigest := fmt.Sprintf("%x", backupFileHash.Sum(nil))

	return sourceFileDigest == backupDigest
}

func (b *Backup) toString() string {
	return toJSON(b)
}

// CreateConfig will create the configuration file based on the Backup struct
func (b *Backup) CreateConfig(path string) {
	outputJSON, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err)
	}
	ioutil.WriteFile(path, outputJSON, os.ModePerm)
}

// LoadConfig will handle loading up the given config file
func (b *Backup) LoadConfig(path string) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	json.Unmarshal(raw, &b)
}

// CreateOrLoadConfig will first check to see if the config file exists in the
// given path, if not it will create it, otherwise it will load it.
func (b *Backup) CreateOrLoadConfig(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Setup some place holders
		b.SourcePath = "Source Root Directory Path"
		b.BackupPath = "Backup Root Directory Path"
		b.TimeBetweenBackups = 24
		// Create the config file
		b.CreateConfig(path)
		// Get the current path that the Executable is running at
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex) + "\\config.json"
		// Open a command prompt and echo a message to let the user know that
		// the config.json file should be updated.
		cmd := "cmd"
		args := []string{"/C", "start", "C:\\Windows\\System32\\cmd.exe", "/k",
			"echo Please update " + exPath + " and re-run the program!"}
		if err := exec.Command(cmd, args...).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	} else {
		b.LoadConfig(path)
	}
}

// Copy file from one directory to another
func (b *Backup) Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func toJSON(b interface{}) string {
	bytes, err := json.Marshal(b)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	return string(bytes)
}

func walkSource(b *Backup) filepath.WalkFunc {
	return func(path string, f os.FileInfo, err error) error {
		backupPath := b.BackupPath + path[len(b.SourcePath):]
		// If the given directory/file does not currently exist then we create
		// or copy it, if it does then only copy it if the MD5 checksum is
		// different from the source file.
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			// If the current path element in the source is a directory then
			// create a directory in the backup path, otherwise copy the file
			// from the source path to the backup path.
			if stat, _ := os.Stat(path); stat.IsDir() {
				os.Mkdir(backupPath, os.ModePerm)
			} else {
				b.Copy(path, backupPath)
			}
		} else {
			if stat, _ := os.Stat(path); !stat.IsDir() {
				if !b.FileChecksum(path, backupPath) {
					b.Copy(path, backupPath)
				}
			}
		}

		return nil
	}

}
