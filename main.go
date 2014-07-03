package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
)

var pngQueue = make(chan string, 5000)
var waitGroup sync.WaitGroup

func init() {
	if _, err := exec.LookPath("pngquant"); err != nil {
		log.Println("'pngquant' command not found in PATH")
		os.Exit(1)
	} else if _, err := exec.LookPath("pngout"); err != nil {
		log.Println("'pngout' command not found in PATH")
		os.Exit(1)
	}
}

func pngWorker(id int) {
	for {
		if element, channelOpen := <-pngQueue; element == "" && !channelOpen {
			log.Println("Queue closed, goroutine #", id, "will terminate")
			waitGroup.Done()
			return
		} else {
			var cmds [2]*exec.Cmd
			var err error
			cmds[0] = exec.Command("pngquant", "--speed", "1", "--ext", ".png", "--force", "256", element)
			cmds[1] = exec.Command("pngout", "-f6", "-kp", "-ks", element)
			for _, cmd := range cmds[:] {
				err = cmd.Run()
				if err != nil {
					log.Println("External command:", err)
				}
			}
		}
	}
}

func walkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Println(err)
		return err
	}
	re := regexp.MustCompile(`.*\.([a-z]+)$`)
	if !info.IsDir() {
		match := re.FindStringSubmatch(info.Name())
		if match != nil {
			fileType := match[1]
			if fileType == "png" {
				pngQueue <- path
			}
		}
	}
	return nil
}

func main() {
	log.SetFlags(log.Ltime)
	dir := flag.String("dir", "/dev/null", "Target directory holding the PNG images")
	cores := flag.Int("cores", runtime.NumCPU(), "Number of cores to use")
	flag.Parse()
	runtime.GOMAXPROCS(*cores)
	log.Println("Starting", *cores, "goroutines")
	for counter := *cores; counter > 0; counter-- {
		go pngWorker(counter)
		waitGroup.Add(1)
	}
	log.Println("Walking directory")
	filepath.Walk(*dir, walkFunc)
	log.Println("Closing queue")
	close(pngQueue)
	log.Println("Waiting for goroutines")
	waitGroup.Wait()
}
