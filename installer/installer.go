package installer

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ShawnEdgell/skaterxl-map-manager/api"
)

var Logger *log.Logger = log.Default()

func InstallMap(mapToInstall api.Map, skaterXLMapsDir string) error {
	if mapToInstall.Modfile.Download.BinaryURL == "" {
		return fmt.Errorf("no download URL found for map %s", mapToInstall.Name)
	}

	tempDir, err := os.MkdirTemp("", "skaterxl-map-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempZipPath := filepath.Join(tempDir, mapToInstall.Modfile.Filename)
	Logger.Printf("Downloading '%s' to '%s' from URL: %s", mapToInstall.Name, tempZipPath, mapToInstall.Modfile.Download.BinaryURL)
	err = downloadFile(tempZipPath, mapToInstall.Modfile.Download.BinaryURL)
	if err != nil {
		return fmt.Errorf("failed to download map: %w", err)
	}

	Logger.Printf("Extracting '%s'...", mapToInstall.Name)

	// Determine the final destination path for the map
	mapDestinationDir := filepath.Join(skaterXLMapsDir, sanitizeFilename(mapToInstall.Name))

	// Create the final destination directory if it doesn't exist
	if _, err := os.Stat(mapDestinationDir); os.IsNotExist(err) {
		err = os.MkdirAll(mapDestinationDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create map destination directory '%s': %w", mapDestinationDir, err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking map destination directory '%s': %w", mapDestinationDir, err)
	}

	// Extract the zip file to a temporary extraction directory
	tempExtractDir := filepath.Join(tempDir, "extracted_zip")
	if err := os.MkdirAll(tempExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create temporary extraction directory: %w", err)
	}

	err = unzip(tempZipPath, tempExtractDir)
	if err != nil {
		return fmt.Errorf("failed to extract map '%s' to temporary location: %w", mapToInstall.Name, err)
	}

	// Check if the extracted content has a single root folder
	singleRootFolder, err := getSingleRootFolder(tempExtractDir)
	if err == nil && singleRootFolder != "" {
		// If there's a single root folder, move its contents directly to the mapDestinationDir
		Logger.Printf("Detected single root folder '%s' in zip. Moving contents to '%s'.", singleRootFolder, mapDestinationDir)
		sourcePath := filepath.Join(tempExtractDir, singleRootFolder)
		err = moveDirContents(sourcePath, mapDestinationDir)
		if err != nil {
			return fmt.Errorf("failed to move contents from single root folder: %w", err)
		}
	} else {
		// Otherwise, move all extracted contents directly to the mapDestinationDir
		Logger.Printf("No single root folder detected or error: %v. Moving all extracted contents to '%s'.", err, mapDestinationDir)
		err = moveDirContents(tempExtractDir, mapDestinationDir)
		if err != nil {
			return fmt.Errorf("failed to move extracted contents: %w", err)
		}
	}

	Logger.Printf("Successfully installed '%s' to '%s'!", mapToInstall.Name, mapDestinationDir)
	return nil
}

func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src, dest string) error {
    r, err := zip.OpenReader(src)
    if err != nil {
        return err
    }
    defer func() {
        if err := r.Close(); err != nil {
            panic(err)
        }
    }()

    os.MkdirAll(dest, 0755)

    extractAndWriteFile := func(f *zip.File) error {
        rc, err := f.Open()
        if err != nil {
            return err
        }
        defer func() {
            if err := rc.Close(); err != nil {
                panic(err)
            }
        }()

        path := filepath.Join(dest, f.Name)
        if !strings.HasPrefix(path, filepath.Clean(dest) + string(os.PathSeparator)) {
            return fmt.Errorf("illegal file path: %s", path)
        }

        if f.FileInfo().IsDir() {
            os.MkdirAll(path, f.Mode())
        } else {
            os.MkdirAll(filepath.Dir(path), f.Mode())
            f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
            if err != nil {
                return err
            }
            defer func() {
                if err := f.Close(); err != nil {
                    panic(err)
                }
            }()

            _, err = io.Copy(f, rc)
            if err != nil {
                return err
            }
        }
        return nil
    }

    for _, f := range r.File {
        err := extractAndWriteFile(f)
        if err != nil {
            return err
        }
    }
    return nil
}

func getZipRootFolder(zipFilePath string) (string, error) {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	if len(r.File) == 0 {
		return "", fmt.Errorf("zip file is empty")
	}

	firstPath := r.File[0].Name
	parts := strings.Split(firstPath, string(os.PathSeparator))
	if len(parts) > 0 && parts[0] != "" {
		return parts[0], nil
	}
	return "", fmt.Errorf("could not determine root folder from zip")
}

func sanitizeFilename(name string) string {
    invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
    for _, char := range invalidChars {
        name = strings.ReplaceAll(name, char, "_")
    }
    return name
}

// getSingleRootFolder checks if the extracted directory contains a single root folder.
// Returns the name of the root folder if found, otherwise an empty string and an error.
func getSingleRootFolder(extractedPath string) (string, error) {
	entries, err := os.ReadDir(extractedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted directory: %w", err)
	}

	var rootFolders []string
	for _, entry := range entries {
		if entry.IsDir() {
			rootFolders = append(rootFolders, entry.Name())
		} else {
			// If there are files directly in the root, it's not a single root folder structure
			return "", fmt.Errorf("files found directly in extracted root")
		}
	}

	if len(rootFolders) == 1 {
		return rootFolders[0], nil
	}
	return "", fmt.Errorf("no single root folder found")
}

// moveDirContents moves all contents (files and subdirectories) from src to dest.
// It performs a copy-then-delete operation, which is more robust across filesystems.
func moveDirContents(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory '%s': %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			err := copyDir(srcPath, destPath)
			if err != nil {
				return fmt.Errorf("failed to copy directory '%s' to '%s': %w", srcPath, destPath, err)
			}
		} else {
			err := copyFile(srcPath, destPath)
			if err != nil {
				return fmt.Errorf("failed to copy file '%s' to '%s': %w", srcPath, destPath, err)
			}
		}
	}

	// Remove the source directory after successful copy
	return os.RemoveAll(src)
}

// copyFile copies a file from src to dest.
func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file '%s': %w", dest, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents from '%s' to '%s': %w", src, dest, err)
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source file info '%s': %w", src, err)
	}

	return os.Chmod(dest, sourceInfo.Mode())
}

// copyDir recursively copies a directory from src to dest.
func copyDir(src, dest string) error {
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to get source directory info '%s': %w", src, err)
	}

	if err := os.MkdirAll(dest, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory '%s': %w", dest, err)
	}

	dirents, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source directory '%s': %w", src, err)
	}

	for _, dirent := range dirents {
		srcPath := filepath.Join(src, dirent.Name())
		destPath := filepath.Join(dest, dirent.Name())

		if dirent.IsDir() {
			err := copyDir(srcPath, destPath)
			if err != nil {
				return err
			}
		} else {
			err := copyFile(srcPath, destPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}