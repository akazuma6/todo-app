package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Task struct {
	ID    string
	Title string
}

var sheetsService *sheets.Service
var spreadsheetID = "19sGTnXqhOkZSIsJZSS_weSvO43FtFceXKNRc2ovLhY0"

func main() {

	ctx := context.Background()
	// credentials.jsonを使って認証
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("credentials.json"))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}
	sheetsService = srv

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")

	// GET / : タスク一覧ページ
	router.GET("/", func(c *gin.Context) {
		// A2からB列の最後までデータを取得（A1,B1は見出しなのでA2から）
		readRange := "Sheet1!A:B"
		resp, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
		if err != nil {
			c.String(http.StatusInternalServerError, "シートからデータを取得できませんでした: %v", err)
			return
		}

		var tasks []Task
		if len(resp.Values) > 1 {
			// resp.Values[1:] でヘッダー行(0番目)をスライスして無視する
			for i, row := range resp.Values[1:] {
				// IDは実際の行番号(A2が2行目)とする
				taskID := strconv.Itoa(i + 2)
				taskTitle := ""
				if len(row) > 1 {
					// タイトルはB列なので、row[1] を参照する
					taskTitle, _ = row[1].(string)
				} else if len(row) > 0 {
					// A列にIDだけ入っていてB列が空の場合
					taskTitle, _ = row[0].(string)
				}

				tasks = append(tasks, Task{ID: taskID, Title: taskTitle})
				fmt.Println("フォームから受け取ったtasks:", tasks)
			}
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"tasks": tasks,
		})
	})

	// POST /new : 新規タスクの作成
	router.POST("/new", func(c *gin.Context) {
		title := c.PostForm("title")
		if title == "" {
			c.String(http.StatusBadRequest, "タイトルは必須です")
			return
		}

		// スプレッドシートの末尾に新しい行を追加
		var vr sheets.ValueRange
		// IDは自動で振られる（行番号になる）ので、タイトルだけを追加
		vr.Values = append(vr.Values, []interface{}{"", title})

		// Sheet1のA列（の最終行の次）にデータを追記
		_, err := sheetsService.Spreadsheets.Values.Append(spreadsheetID, "Sheet1!A1", &vr).ValueInputOption("USER_ENTERED").Do()
		if err != nil {
			c.String(http.StatusInternalServerError, "シートへの書き込みに失敗しました: %v", err)
			return
		}

		c.Redirect(http.StatusFound, "/")
	})

	router.Run(":8080")
}
