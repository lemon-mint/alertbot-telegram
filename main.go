package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lemon-mint/godotenv"
	"github.com/patrickmn/go-cache"
	"golang.org/x/crypto/acme/autocert"
)

var apiBaseURL string

type saystate struct {
	ChatID int
	UserID int
	state  string
}

var statedb = cache.New(10*time.Minute, 3*time.Minute)

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
			{
				Command:     "/timeunix",
				Description: "Displays the current Unix time.",
			},
		},
	})
	//fmt.Println(string(data))
	resp, _ := http.Post(
		apiBaseURL+"/setMyCommands",
		"application/json",
		bytes.NewReader(data),
	)
	ioutil.ReadAll(resp.Body)
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

type webHookSendStyledMessageResponse struct {
	Method string `json:"method"`
	ChatID int    `json:"chat_id"`
	Text   string `json:"text"`
	Mode   string `json:"parse_mode"`
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
		stateresp, ok := runState(request.Message.Text, *request)
		if ok {
			if stateresp.Type == "string" {
				return c.JSON(200, webHookSendMessageResponse{
					Method: "sendMessage",
					ChatID: request.Message.Chat.ID,
					Text:   stateresp.StringBody,
				})
			} else if stateresp.Type == "image" {
				return c.JSON(200, webHookSendPhotoResponse{
					Method: "sendPhoto",
					ChatID: request.Message.Chat.ID,
					Photo:  stateresp.ImageURL,
				})
			} else if stateresp.Type == "stringReply" {
				return c.JSON(200, webHookSendMessageReplyResponse{
					Method:  "sendMessage",
					ChatID:  request.Message.Chat.ID,
					Text:    stateresp.StringBody,
					ReplyID: request.Message.MessageID,
				})
			} else if stateresp.Type == "stringstyle" {
				return c.JSON(200, webHookSendStyledMessageResponse{
					Method: "sendMessage",
					ChatID: request.Message.Chat.ID,
					Text:   stateresp.StringBody,
					Mode:   stateresp.StyleFormat,
				})
			}
		}
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
			} else if resp.Type == "stringstyle" {
				return c.JSON(200, webHookSendStyledMessageResponse{
					Method: "sendMessage",
					ChatID: request.Message.Chat.ID,
					Text:   resp.StringBody,
					Mode:   resp.StyleFormat,
				})
			}
		}
	}
	return c.String(200, "Sorry")
}

type tgResponse struct {
	Type        string
	StringBody  string
	ImageURL    string
	StyleFormat string
	Body        interface{}
}

func runBotCmd(cmd string, ctx webhookEventRequest) tgResponse {
	if cmd == "/start" {
		return tgResponse{
			Type:       "string",
			StringBody: "Hello, User!",
		}
	}
	if cmd == "/timeunix" {
		return tgResponse{
			Type:       "string",
			StringBody: strconv.Itoa(int(time.Now().UTC().Unix())),
		}
	}
	if cmd == "/ping" {
		return tgResponse{
			Type:        "stringstyle",
			StringBody:  "<i>Pong!</i>",
			StyleFormat: "HTML",
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
	if cmd == "/say" {
		go statedb.Set("saystate_"+strconv.Itoa(ctx.Message.Chat.ID), saystate{
			ChatID: ctx.Message.Chat.ID,
			UserID: ctx.Message.From.ID,
			state:  "input",
		}, cache.DefaultExpiration)
		return tgResponse{
			Type:       "string",
			StringBody: "Please enter a sentence to say 😀",
		}
	}
	if cmd == "/dump" {
		go statedb.Set("_dump_"+strconv.Itoa(ctx.Message.Chat.ID), saystate{
			ChatID: ctx.Message.Chat.ID,
			UserID: ctx.Message.From.ID,
			state:  "input",
		}, cache.DefaultExpiration)
		return tgResponse{
			Type:       "string",
			StringBody: "Please enter a sentence to dump 🍳",
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

func runState(text string, ctx webhookEventRequest) (tgResponse, bool) {
	state, ok := statedb.Get("saystate_" + strconv.Itoa(ctx.Message.Chat.ID))
	if ok {
		if state.(saystate).state == "input" &&
			state.(saystate).ChatID == ctx.Message.Chat.ID &&
			state.(saystate).UserID == ctx.Message.From.ID {
			statedb.Delete("saystate_" + strconv.Itoa(ctx.Message.Chat.ID))
			go sendPlainMessage(ctx.Message.Chat.ID, text, int64(time.Millisecond*400), 0)
			return tgResponse{
				Type:       "string",
				StringBody: "I understand, wait a moment 🤔",
			}, true
		}
	}
	state, ok = statedb.Get("_dump_" + strconv.Itoa(ctx.Message.Chat.ID))
	if ok {
		if state.(saystate).state == "input" &&
			state.(saystate).ChatID == ctx.Message.Chat.ID &&
			state.(saystate).UserID == ctx.Message.From.ID {
			statedb.Delete("_dump_" + strconv.Itoa(ctx.Message.Chat.ID))
			data, _ := json.Marshal(ctx)
			return tgResponse{
				Type:       "string",
				StringBody: string(data),
			}, true
		}
	}
	return tgResponse{}, false
}

func sendPlainMessage(chatID int, msg string, delay int64, postDelay int64) {
	jsondata, err := json.Marshal(
		struct {
			ChatID int    `json:"chat_id"`
			Text   string `json:"text"`
		}{
			ChatID: chatID,
			Text:   msg,
		},
	)
	if err == nil {
		if delay != 0 {
			time.Sleep(time.Duration(delay))
		}
		http.Post(apiBaseURL+"/sendMessage", "application/json", bytes.NewReader(jsondata))
		if postDelay != 0 {
			time.Sleep(time.Duration(postDelay))
		}
	}
}
