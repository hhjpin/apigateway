package main

import (
	"fmt"
	"strings"
	"net/smtp"
)

func CollectPanic(body string) {
	auth := smtp.PlainAuth("", "service@hengha.ren", "HengHaRen1010", "smtp.exmail.qq.com")
	to := []string{"riposa@hengha.ren"}
	nickname := "drop_shipping"
	user := "service@hengha.ren"
	subject := "drop_shipping err mail"
	contentType := "Content-Type: text/plain; charset=UTF-8"
	msg := []byte("To: " + strings.Join(to, ",") + "\r\nFrom: " + nickname +
		"<" + user + ">\r\nSubject: " + subject + "\r\n" + contentType + "\r\n\r\n" + body)
	err := smtp.SendMail("smtp.exmail.qq.com:25", auth, user, to, msg)
	if err != nil {
		fmt.Printf("send mail error: %v", err)
	}
}
