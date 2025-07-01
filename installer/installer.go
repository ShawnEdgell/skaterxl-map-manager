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

	extractedBaseDir, err := getZipRootFolder(tempZipPath)
	if err != nil {
		Logger.Printf("Warning: Could not determine zip root folder, using map name for destination: %v", err)
		extractedBaseDir = mapToInstall.Name
	}
    extractedBaseDir = sanitizeFilename(extractedBaseDir)


	destinationPath := filepath.Join(skaterXLMapsDir, extractedBaseDir)

	if _, err := os.Stat(destinationPath); os.IsNotExist(err) {
		err = os.MkdirAll(destinationPath, 0755)
		if err != nil {
			return fmt.Errorf("failed to create destination directory '%s': %w", destinationPath, err)
		}
	} else if err != nil {
		return fmt.Errorf("error checking destination directory '%s': %w", destinationPath, err)
	}

	err = unzip(tempZipPath, destinationPath)
	if err != nil {
		return fmt.Errorf("failed to extract map '%s' to '%s': %w", mapToInstall.Name, destinationPath, err)
	}

	Logger.Printf("Successfully installed '%s' to '%s'!", mapToInstall.Name, destinationPath)
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