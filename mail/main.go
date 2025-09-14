package main

import "net/smtp"

func main() {
	// 设置认证信息
	auth := smtp.PlainAuth("", "user@example.com", "password", "smtp.example.com")

	// 设置邮件内容
	msg := []byte("To: recipient@example.com\r\n" +
		"Subject: Hello!\r\n" +
		"\r\n" +
		"This is the email body.\r\n")

	// 发送邮件
	err := smtp.SendMail("smtp.example.com:587", auth, "sender@example.com", []string{"recipient@example.com"}, msg)
	if err != nil {
		panic(err)
	}
	//InitConfig()
}
