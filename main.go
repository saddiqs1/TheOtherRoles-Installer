package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	amongUsPath, err := findAmongUsDir()
	if err != nil {
		fmt.Print(err)
		return
	}

	zipFile, err := downloadFile()
	if err != nil {
		fmt.Print(err)
		return
	}

	moddedPath := filepath.Join(filepath.Dir(amongUsPath), "Among Us Modded")
	err = deleteDir(moddedPath)
	if err != nil {
		fmt.Print(err)
		return
	}

	err = copyDir(amongUsPath, moddedPath)
	if err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("Among Us folder copied to: %s\n", moddedPath)

	err = extractZip(zipFile, moddedPath)
	if err != nil {
		fmt.Print(err)
		return
	}

	err = os.Remove(zipFile)
	if err != nil {
		fmt.Printf("Error deleting zip file: %v\n", err)
		return
	}
	fmt.Println("Zip file deleted successfully.")

	exePath := filepath.Join(moddedPath, "Among Us.exe")
	err = createDesktopShortcut(exePath, "Among Us Modded")
	if err != nil {
		fmt.Printf("Error creating desktop shortcut: %v\n", err)
		return
	}
	fmt.Println("Desktop shortcut created successfully.")

	err = deleteAppDataFolder()
	if err != nil {
		fmt.Print(err)
		return
	}
}

func findAmongUsDir() (string, error) {
	vdfPath := `C:\Program Files (x86)\Steam\steamapps\libraryfolders.vdf`

	file, err := os.Open(vdfPath)
	if err != nil {
		return "", fmt.Errorf("error opening file %q: %v", vdfPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var currentLibraryPath string
	found := false
	inAppsBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, `"path"`) {
			parts := strings.Split(line, `"`)
			if len(parts) >= 4 {
				currentLibraryPath = parts[3]
			}
		}

		if strings.Contains(line, `"apps"`) {
			inAppsBlock = true
			continue
		}

		if inAppsBlock {
			if strings.Contains(line, `"945360"`) {
				found = true
				break
			}
			if strings.HasPrefix(line, "}") {
				inAppsBlock = false
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	if found && currentLibraryPath != "" {
		amongUsPath := filepath.Join(currentLibraryPath, "steamapps", "common", "Among Us")
		return amongUsPath, nil
	}

	return "", fmt.Errorf("could not locate Among Us installation in the libraryfolders.vdf file.")

}

func downloadFile() (string, error) {
	url := "https://github.com/Eisbison/TheOtherRoles/releases/latest/download/TheOtherRoles.zip"
	filePath := "TheOtherRoles.zip"

	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %v", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving file: %v", err)
	}

	fmt.Printf("File downloaded successfully to %s\n", filePath)
	return filePath, nil
}

func deleteDir(src string) error {
	if _, err := os.Stat(src); err == nil {
		fmt.Println("Existing 'Among Us Modded' folder found, deleting...")
		err = os.RemoveAll(src)
		if err != nil {
			return fmt.Errorf("error deleting existing folder: %v", err)
		}
	}

	return nil
}

func copyDir(src string, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s is not a directory", src)
	}

	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

func extractZip(zipPath string, destination string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	fmt.Println("Mods extracted successfully into the new folder.")
	return nil
}

func createDesktopShortcut(target, shortcutName string) error {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE environment variable not set")
	}
	desktopPath := filepath.Join(userProfile, "Desktop")
	shortcutPath := filepath.Join(desktopPath, shortcutName+".lnk")
	workingDir := filepath.Dir(target)
	vbsContent := fmt.Sprintf(`Set oWS = WScript.CreateObject("WScript.Shell")
sLinkFile = "%s"
Set oLink = oWS.CreateShortcut(sLinkFile)
oLink.TargetPath = "%s"
oLink.WorkingDirectory = "%s"
oLink.Save
`, shortcutPath, target, workingDir)
	tmpFile, err := os.CreateTemp("", "createshortcut-*.vbs")
	if err != nil {
		return fmt.Errorf("failed to create temporary VBS file: %v", err)
	}
	vbsPath := tmpFile.Name()
	_, err = tmpFile.WriteString(vbsContent)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary VBS file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(vbsPath)
	cmd := exec.Command("cscript", "//nologo", vbsPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute VBS script: %v, output: %s", err, output)
	}
	return nil
}

func deleteAppDataFolder() error {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return fmt.Errorf("USERPROFILE environment variable not set")
	}

	targetDir := filepath.Join(userProfile, "AppData", "LocalLow", "Innersloth")

	fmt.Println("Deleting directory:", targetDir)

	err := os.RemoveAll(targetDir)
	if err != nil {
		return fmt.Errorf("error deleting directory: %v", err)
	}

	fmt.Println("AppData directory deleted successfully.")
	return nil
}
