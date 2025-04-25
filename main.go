package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// log to file and console
type MultiWriter struct {
	writers []io.Writer
}

func (t *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return
}

// Config представляет основную конфигурацию
type Config struct {
	TargetsPath     string   //folder with target template
	ModulesPath     string   //folder with module template
	OutputPath      string   //Path with Netbios Name witch use to save result collection
	OutputPathZip   string   //ZIP the  `OutputPath` folder
	SourcePath      string   //The name of Disk (Default C:\\) ??
	MaxParallel     int      //Count worket
	needTargetName  []string //The array with name of Target template
	needModulesName []string //The array with name of Modules template

	NeedTargetNameString string `yaml:"need_target_name_string"` //The names of Targets wich need use
	NeedModuleNameString string `yaml:"need_module_name_string"` //The names of Modules  wich need use

	//zip arch
	Password string `yaml:"password"` //Password of zip
	NeedZip  bool   `yaml:"zip"`      //Bool need password ZIP or NOT

	//folder to rawcopy
	PathToRawCopy  string `yaml:"path_to_rawcopy"` //Path to RAW-COPY
	PathToAllTools string `yaml:"path_to_all_bin"` //Path to other binary

	//remove after execute
	RemoveAfterExecute bool `yaml:"remove_after_execute"` //Remove executable and template file or not

	//send to server
	SendToServer     bool   `yaml:"send_to_server"`
	SendToServerIP   string `yaml:"send_to_server_ip"`
	SendToServerURL  string `yaml:"send_to_server_url"`
	SendToServerPort int    `yaml:"send_to_server_port"`

	//send to own
	SendToOwnServer       bool   `yaml:"send_to_own"`
	SendToOwnServerIP     string `yaml:"send_to_own_server_ip"`
	SendToOwnServerPORT   int    `yaml:"send_to_own_server_port"`
	SendToOwnServerFolder string `yaml:"name_folder"`

	//remove after send
	RemoveAfterSend bool `yaml:"remove_after_send"`
}

// FileInfo хранит информацию о найденном файле
type FileInfo struct {
	SourcePath string
	TargetPath string
	Category   string
	Name       string
}

// Collector управляет сбором артефактов
type Collector struct {
	config  Config
	targets []Target
	modules []Module
	wg      sync.WaitGroup
	rawCopy RawCopy
}

func NewCollector(config Config) (*Collector, error) {
	c := &Collector{
		config: config,
	}

	// Загружаем targets
	targets, err := c.loadTargets()
	if err != nil {
		return nil, fmt.Errorf("error loading targets: %v", err)
	}
	c.targets = targets

	// Загружаем modules
	modules, err := c.loadModules()
	if err != nil {
		return nil, fmt.Errorf("error loading modules: %v", err)
	}

	c.modules = modules

	rawCopy, err := NewRawCopy(config.PathToRawCopy)
	c.rawCopy = *rawCopy

	return c, nil
}

func loadConfig() (*Config, error) {
	// Устанавливаем значения по умолчанию

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	needModulesNameString := "Default"
	needTargetsNameString := "Default"
	needTargetsName := strings.Split(needTargetsNameString, ",")
	needModulesName := strings.Split(needModulesNameString, ",")

	config := &Config{
		TargetsPath:          ".\\Targets",
		ModulesPath:          ".\\Modules",
		OutputPath:           ".\\" + string(hostname),
		OutputPathZip:        ".\\" + string(hostname) + ".zip",
		SourcePath:           "C:/", // Измените на нужный путь,
		MaxParallel:          runtime.NumCPU(),
		needTargetName:       needTargetsName,
		needModulesName:      needModulesName,
		NeedModuleNameString: needModulesNameString,
		NeedTargetNameString: needTargetsNameString,
	}

	data, err := ioutil.ReadFile("settings.yaml")
	if err != nil {
		return config, err // Возвращаем конфиг по умолчанию при ошибке чтения
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return config, err
	}

	config.needModulesName = strings.Split(config.NeedModuleNameString, ",")
	config.needTargetName = strings.Split(config.NeedTargetNameString, ",")

	return config, nil
}

func main() {

	fmt.Println("Start MARS ARTEFACT COLLECTOR")
	startPicture := `
	`
	fmt.Println(startPicture)
	start := time.Now()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error parse config: %v", err)
	}

	collector, err := NewCollector(*config)
	if err != nil {
		log.Fatalf("Error creating collector: %v", err)
	}

	os.Mkdir(config.OutputPath, 0777)
	// Открываем log-файл для записи (создаем, если не существует)
	file, err := os.OpenFile(config.OutputPath+"\\kulac.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Создаем MultiWriter, который будет писать и в консоль, и в файл
	_ = &MultiWriter{
		writers: []io.Writer{
			os.Stdout,
			file,
		},
	}
	// Переопределяем стандартный вывод
	stdout := os.Stdout
	os.Stdout = os.NewFile(uintptr(file.Fd()), "stdout")
	defer func() {
		os.Stdout = stdout
	}()

	// Запускаем сбор целей
	fmt.Println("Starting target collection...")
	if err := collector.CollectTargets(); err != nil {
		log.Fatalf("Error collecting targets: %v", err)
	}

	// Запускаем выполнение модулей
	fmt.Println("Starting module execution...")
	if err := collector.ExecuteModules(); err != nil {
		log.Fatalf("Error executing modules: %v", err)
	}

	fmt.Println("Collection completed successfully")
	elapsed := time.Since(start)
	fmt.Printf("Время выполнения: %s\n", elapsed)

	if collector.config.NeedZip {
		fmt.Println("Start Zipping")
		zipDirectory(config.OutputPath, config.OutputPathZip, config.Password)
		fmt.Println("End Zipping")
	}

	if collector.config.SendToServer {
		fmt.Println("Start send to web-server")
		_, err = collector.uploadFileMultipart()
		if err != nil {
			fmt.Println("Успешно отправлено на сервер")
		}
		if collector.config.RemoveAfterSend {
			os.RemoveAll(collector.config.OutputPathZip)
		}
	}

	if collector.config.SendToOwnServer {
		fmt.Println("Start send to own-server")
		path := ""
		if collector.config.NeedZip {
			path = collector.config.OutputPathZip
		} else {
			path = collector.config.OutputPath
		}

		webdavURL := "https://81.177.34.211/remote.php/webdav/MID_upload/"
		webdavURL_with_files := webdavURL + strings.Replace(path, ".\\", "", -1)

		err := collector.uploadToWebDAV(webdavURL_with_files, path)

		if err != nil {
			fmt.Printf("Error uploading file: %v\n", err)
			os.Exit(1)
		}
		if collector.config.RemoveAfterSend {
			os.RemoveAll(collector.config.OutputPathZip)
		}
	}

	if collector.config.RemoveAfterExecute {
		fmt.Println("Start remove the executable files")
		os.RemoveAll(collector.config.TargetsPath)
		os.RemoveAll(collector.config.ModulesPath)
		os.RemoveAll(collector.config.OutputPath)
		os.RemoveAll(collector.config.PathToRawCopy)
		os.RemoveAll(collector.config.PathToAllTools)
		os.RemoveAll("settings.yaml")
		os.RemoveAll("start.bat")
		os.RemoveAll("kulac.exe")

		selfDeleteWindows()
	}
}
