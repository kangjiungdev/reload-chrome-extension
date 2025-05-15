# Reload chrome extension

chrome extension reloader program made with golang

### 확장프로그램 background.js 예시

```js
const serverPort = 서버포트;

let lastTimestamp = null;
let waiting = false;

async function checkChangeEventRequest() {
  if (waiting) return;
  waiting = true;

  try {
    const res = await fetch(`http://localhost:${serverPort}/`, {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    });
    if (res.ok) {
      const json = await res.json();
      const latestDate = Object.values(json)
        .map((dateStr) => new Date(dateStr))
        .reduce((max, curr) => (curr > max ? curr : max));

      if (latestDate !== lastTimestamp) {
        console.log("변경 감지됨. 리로드 실행");
        lastTimestamp = latestDate;
        chrome.runtime.reload();
      } else {
        console.log("변경 없음");
      }
    } else {
      console.log("서버 응답 실패");
    }
  } catch (err) {
    console.error("요청 실패:", err);
  }

  waiting = false;
}

// 주기적으로 체크
setInterval(checkChangeEventRequest, 2000);
```
