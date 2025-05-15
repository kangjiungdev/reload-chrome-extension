package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	lastModified = map[string]time.Time{}
	debounceDur  = 300 * time.Millisecond // 이 안에 또 발생하면 무시
)

func isDebounced(path string) bool {
	now := time.Now()
	if t, ok := lastModified[path]; ok && now.Sub(t) < debounceDur {
		return true
	}
	lastModified[path] = now
	return false
}

func getPath() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("확장 프로그램 폴더 경로를 입력해 주세요: ")
	path, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("error:", err)
	}
	path = strings.TrimSpace(path)
	fmt.Println(path)
	if strings.HasPrefix(path, "../") {
		return path
	}
	path = fmt.Sprintf("../%s", path)
	fmt.Println(path)
	return path
}

func main() {
	dir := getPath()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close() // 어차피 함수 끝나면 프로그램 종료니까 안넣어도 되긴 하다만 명시적으로 닫아주는게 좀 깔끔해 보여서 넣음

	err = watcher.Add(dir)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		checkFolderEvent(dir, watcher, res, req)
	})
	http.ListenAndServe(":1234", nil)
	fmt.Println("server start")
}

func checkFolderEvent(dir string, watcher *fsnotify.Watcher, res http.ResponseWriter, req *http.Request) {
	accept := req.Header.Get("Accept")

	if !strings.Contains(accept, "application/json") || req.Method != http.MethodGet {
		return
	}

	log.Println("Watching directory:", dir)

	for {
		select {
		case event := <-watcher.Events:
			// vsc 프리티어 때문에 2번 중첩되어서 저장(파일 저장 -> 포마트 -> 포마트 된 코드 저장)되는거 무시
			if !isDebounced(event.Name) {
				// 수정, 생성, 삭제 등 모든 이벤트 감지

				log.Printf("이벤트 발생: %s %s\n", event.Op, event.Name)

				// 수정 이벤트만 필터링
				if event.Op&fsnotify.Write != 0 {
					log.Println("파일 수정됨:", event.Name)
				}
				res.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(res).Encode(lastModified); err != nil {
					http.Error(res, err.Error(), http.StatusInternalServerError)
					fmt.Println("encoder err")
					return
				}
				fmt.Println("responed")
				return
			}

		case err := <-watcher.Errors:
			log.Println("에러:", err)
		}
	}
}
