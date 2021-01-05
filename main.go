package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lemon-mint/godotenv"
	"golang.org/x/crypto/acme/autocert"
)

var apiBaseURL string

func main() {
	godotenv.Load()

	apiBaseURL = os.Getenv("TELEGRAM_API_SERVER_URL") + "/bot" + os.Getenv("TELEGRAM_API_KEY")

	e := echo.New()
	e.AutoTLSManager.Cache = autocert.DirCache(".cache")
	e.AutoTLSManager.HostPolicy = autocert.HostWhitelist(os.Getenv("PUBLIC_HOSTNAME"))
	e.Use(middleware.Recover())
	e.POST("/api/webhook/:hookid", webhook)
	http.Get(apiBaseURL + "/setWebhook?url=" + url.QueryEscape(os.Getenv("PUBLIC_URL")+"/api/webhook/telegram"))
	data, _ := json.Marshal(struct {
		Commands []struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		} `json:"commands"`
	}{
		Commands: []struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		}{
			{
				Command:     "/ping",
				Description: "Ping! Pong!",
			},
			{
				Command:     "/say",
				Description: "/say <text>",
			},
		},
	})
	//fmt.Println(string(data))
	http.Post(
		apiBaseURL+"/setMyCommands",
		"application/json",
		bytes.NewReader(data),
	)
	e.Logger.Fatal(e.StartAutoTLS(":13230"))
}

type webhookEventRequest struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

type webHookSendMessageResponse struct {
	Method string `json:"method"`
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
}

type webHookSendMessageReplyResponse struct {
	Method  string `json:"method"`
	ChatID  int    `json:"chat_id"`
	Text    string `json:"text"`
	ReplyID int    `json:"reply_to_message_id"`
}

type webHookSendPhotoResponse struct {
	Method string `json:"method"`
	ChatID int    `json:"chat_id"`
	Photo  string `json:"photo"`
}

func webhook(c echo.Context) error {
	hookid := c.Param("hookid")
	//fmt.Println(hookid)
	request := new(webhookEventRequest)
	c.Bind(request)
	fmt.Println(*request)
	if hookid == "telegram" {
		if strings.HasPrefix(request.Message.Text, "/") {
			resp := runBotCmd(request.Message.Text, *request)
			if resp.Type == "string" {
				return c.JSON(200, webHookSendMessageResponse{
					Method: "sendMessage",
					ChatID: request.Message.Chat.ID,
					Text:   resp.StringBody,
				})
			} else if resp.Type == "image" {
				return c.JSON(200, webHookSendPhotoResponse{
					Method: "sendPhoto",
					ChatID: request.Message.Chat.ID,
					Photo:  resp.ImageURL,
				})
			} else if resp.Type == "stringReply" {
				return c.JSON(200, webHookSendMessageReplyResponse{
					Method:  "sendMessage",
					ChatID:  request.Message.Chat.ID,
					Text:    resp.StringBody,
					ReplyID: request.Message.MessageID,
				})
			}
		}
	}
	return c.String(200, "Sorry")
}

type tgResponse struct {
	Type       string
	StringBody string
	ImageURL   string
	Body       interface{}
}

func runBotCmd(cmd string, ctx webhookEventRequest) tgResponse {
	if cmd == "/start" {
		return tgResponse{
			Type:       "string",
			StringBody: "Hello, User!",
		}
	}
	if cmd == "/ping" {
		return tgResponse{
			Type:       "string",
			StringBody: "Pong!",
		}
	}
	if cmd == "/notrobot" {
		return tgResponse{
			Type:     "image",
			ImageURL: "AgACAgQAAxkDAAOVX_RkDAxZRHR5u5SHjc3XkHC9SbEAAoOrMRs3-KVThAcojU-4vha7dMwqXQADAQADAgADeQADpJ4AAh4E",
		}
	}
	if cmd == "/leave" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			http.Get(apiBaseURL + "/leaveChat?chat_id=" + strconv.Itoa(ctx.Message.Chat.ID))
		}()
		return tgResponse{
			Type:       "string",
			StringBody: "Thank you for using!",
		}
	}
	if strings.HasPrefix(cmd, "/say ") && len(strings.SplitN(cmd, " ", 2)) == 2 {
		return tgResponse{
			Type:       "string",
			StringBody: strings.SplitN(cmd, " ", 2)[1],
		}
	}
	if strings.HasPrefix(cmd, "/replyme ") && len(strings.SplitN(cmd, " ", 2)) == 2 {
		return tgResponse{
			Type:       "stringReply",
			StringBody: strings.SplitN(cmd, " ", 2)[1],
		}
	}
	return tgResponse{
		Type:       "string",
		StringBody: "Unknown Command\n.·´¯`(>▂<)´¯`·.",
	}
}
