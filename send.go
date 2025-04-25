package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (c *Collector) uploadFileMultipart() (*http.Response, error) {

	path := ""

	if c.config.NeedZip {
		path = c.config.OutputPathZip
	} else {
		path = c.config.OutputPath
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	// Reduce number of syscalls when reading from disk.
	bufferedFileReader := bufio.NewReader(f)
	defer f.Close()

	// Create a pipe for writing from the file and reading to
	// the request concurrently.
	bodyReader, bodyWriter := io.Pipe()
	formWriter := multipart.NewWriter(bodyWriter)

	// Store the first write error in writeErr.
	var (
		writeErr error
		errOnce  sync.Once
	)
	setErr := func(err error) {
		if err != nil {
			errOnce.Do(func() { writeErr = err })
		}
	}
	go func() {
		partWriter, err := formWriter.CreateFormFile("file", path)
		setErr(err)
		_, err = io.Copy(partWriter, bufferedFileReader)
		setErr(err)
		setErr(formWriter.Close())
		setErr(bodyWriter.Close())
	}()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:%s/%s", c.config.SendToServerIP, strconv.Itoa(c.config.SendToServerPort), c.config.SendToServerURL), bodyReader)
	if err != nil {
		fmt.Errorf("Error create requests with info: %v\n", err)
		return nil, err
	}
	req.Header.Add("Content-Type", formWriter.FormDataContentType())

	// This operation will block until both the formWriter
	// and bodyWriter have been closed by the goroutine,
	// or in the event of a HTTP error.
	resp, err := http.DefaultClient.Do(req)

	if writeErr != nil {
		fmt.Errorf("Error send requests with info: %v\n", err)
		return nil, writeErr
	}

	return resp, err
}

// /NameIncident - name folder in owncloud
// NameFile - name file in owncloud
func (c *Collector) uploadFileToOwn() error {

	path := ""

	if c.config.NeedZip {
		path = c.config.OutputPathZip
	} else {
		path = c.config.OutputPath
	}

	// Проверяем доступность сервера
	url := fmt.Sprintf("http://%s:%s/webdav/%s/%s", c.config.SendToOwnServerIP, strconv.Itoa(c.config.SendToOwnServerPORT), c.config.SendToOwnServerFolder, strings.Replace(path, ".\\", "", 1))

	// Открываем файл
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Получаем размер файла
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %v", err)
	}

	// Создаем PUT запрос
	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 10 * time.Minute, // Максимальное время загрузки - 10 минут
	}

	// Создаем канал для отслеживания прогресса
	done := make(chan bool)
	go func() {
		fmt.Println("Upload started...")
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				fmt.Println("Still uploading...")
			}
		}
	}()

	// Выполняем запрос
	resp, err := client.Do(req)
	done <- true // Сигнализируем о завершении загрузки

	if err != nil {
		if os.IsTimeout(err) {
			return fmt.Errorf("upload timeout after 10 minutes")
		}
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Collector) uploadToWebDAV(url, filePath string) error {

	username := "KULAC"
	password := "V9o[#)587Ej2"

	// Открываем файл для загрузки
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %v", err)
	}
	defer file.Close()

	// Читаем содержимое файла
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}

	// Создаем кастомный транспорт с отключенной проверкой TLS
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Создаем PUT запрос
	req, err := http.NewRequest("PUT", url, &buf)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}

	// Устанавливаем базовую аутентификацию
	req.SetBasicAuth(username, password)

	// Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("сервер вернул ошибку: %s", resp.Status)
	}

	return nil
}
