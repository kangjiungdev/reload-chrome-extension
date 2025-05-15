package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
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

func checkBigErr(err error) {
	if err != nil {
		log.Fatal("error:", err)
	}
}

func getSetting() (string, string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("확장 프로그램 폴더 경로를 입력해 주세요: ")
	path, err := reader.ReadString('\n')
	checkBigErr(err)
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "../") {
		path = "../" + path
	}
	fmt.Println(path)
	fmt.Print("서버 실행 포트를 입력해 주세요: ")
	name, err := reader.ReadString('\n')
	checkBigErr(err)
	name = strings.TrimSpace(name)
	return path, name
}

func main() {
	dir, port := getSetting()
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
	// 서버 실행 가능한지 미리 확인. 이렇게 안하면 server start on이 서버가 실행됐을 때만 출력되게 할 수 없음. Serve 실행되면 서버 요청 처리하느라 밑에 코드까지 도달 못함.
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("error: 올바른 포트를 입력해 주세요")
		return
	}
	fmt.Println("server start on :" + port)

	// 실제 서버 실행
	err = http.Serve(ln, nil)
	if err != nil {
		fmt.Println("server stopped with error:", err)
	}
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
