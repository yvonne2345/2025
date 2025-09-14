package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/xuri/excelize/v2"
	"html/template"
	"io/ioutil"
	"net/smtp"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type MailContent struct {
	Username string
	Orders   []string
	Deadline string
}

const (
	smtpServer = "smtp.example.com:587"
	username   = "user@example.com"
	password   = "yourpassword"
)

func demo() {
	// 1. 准备模板数据
	data := MailContent{
		Username: "李四",
		Orders:   []string{"笔记本电脑", "无线耳机", "智能手表"},
		Deadline: "2025-09-30",
	}

	// 2. 渲染HTML模板
	htmlBody := renderHTML1("template/email.html", data)

	// 3. 构建MIME邮件 - 带Excel附件
	// 注意：这里的excelFilePath需要是实际存在的Excel文件路径
	excelFilePath := "template/FinOps.xlsx" // 修改为实际的Excel文件路径
	msg := buildEmail1(
		"user@example.com",
		"receiver@example.com",
		"您的订单确认通知",
		htmlBody,
		excelFilePath,
		MailExcelDataInfo{},
	)

	// 4. 发送邮件
	auth := smtp.PlainAuth("", username, password, strings.Split(smtpServer, ":")[0])
	if err := smtp.SendMail(smtpServer, auth, username, []string{"receiver@example.com"}, msg); err != nil {
		fmt.Printf("邮件发送失败: %v\n", err)
	} else {
		fmt.Println("邮件发送成功")
	}
}

func renderHTML1(tplPath string, data MailContent) string {
	tmpl := template.Must(template.ParseFiles(tplPath))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// buildEmail1 构建带HTML正文和可选Excel附件的邮件
func buildEmail1(from, to, subject, html string, excelFilePath string, data MailExcelDataInfo) []byte {
	// 创建一个唯一的边界
	boundary := "-=+XferoMailBoundary+="
	var buffer bytes.Buffer

	// 邮件头部
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", boundary),
	}
	for k, v := range headers {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	buffer.WriteString("\r\n")

	// HTML正文部分
	buffer.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buffer.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buffer.WriteString("\r\n")
	buffer.WriteString(html)
	buffer.WriteString("\r\n")

	// 如果提供了Excel文件路径，则添加附件
	if excelFilePath != "" {
		var fileName string
		var encodedFile []byte
		var err error

		// 如果提供了数据，则创建带数据的Excel附件
		if (reflect.DeepEqual(data, MailExcelDataInfo{})) {
			fileName, encodedFile, err = CreateExcelAttachmentWithData(excelFilePath, data)
		} else {
			// 否则，创建普通的Excel附件
			fileName, encodedFile, err = ConstructAttachment1(excelFilePath)
		}

		if err == nil {
			buffer.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buffer.WriteString("Content-Type: application/vnd.ms-excel\r\n")
			buffer.WriteString("Content-Transfer-Encoding: base64\r\n")
			buffer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", fileName))
			buffer.WriteString("\r\n")
			buffer.Write(encodedFile)
			buffer.WriteString("\r\n")
		}
	}

	// 结束边界
	buffer.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return buffer.Bytes()
}

func CreateExcelAttachmentWithData3(templatePath string, data MailExcelDataInfo) (string, []byte, error) {
	// 1. 读取模板文件
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", nil, fmt.Errorf("打开模板失败: %w", err)
	}

	// 2-1、插入OverWeekExcelData数据
	startRow := 1
	headers := []string{"系统名称", "应用名称", "部署组名", "CPU Container(%)", "CPU Pod(%)", "内存 Container(%)", "内存 Pod(%)", "建议"}
	for colIndex, header := range headers {
		colName, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return "", nil, fmt.Errorf("获取列名失败: %v", err)
		}
		col := colName + fmt.Sprintf("%d", startRow)
		if err := f.SetCellValue("CPU资源使用率超过阈值", col, header); err != nil {
			return "", nil, fmt.Errorf("设置表头失败: %v", err)
		}
	}

	for i, item := range data.OverWeekExcelData {
		row := startRow + i + 1
		rowData := []interface{}{
			item.SystemName,
			item.ApplicationName,
			item.GroupName,
			item.ContainerCpuAvg,
			item.PodCpuAvg,
			item.ContainerMemAvg,
			item.PodMemAvg,
			item.Recommend,
		}
		for colIndex, cellData := range rowData {
			colName, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return "", nil, fmt.Errorf("获取列名失败: %v", err)
			}
			col := colName + fmt.Sprintf("%d", row)
			if err := f.SetCellValue("CPU资源使用率超过阈值", col, cellData); err != nil {
				return "", nil, fmt.Errorf("写入数据失败: %v", err)
			}
		}
	}

	// 2-2、处理BelowWeekExcelData（CPU资源使用率低于阈值）
	startBelowRow := 1
	belowHeaders := []string{"系统名称", "应用名称", "组名", "CPU Container(%)", "CPU Pod(%)", "内存 Container(%)", "内存 Pod(%)", "建议"}
	for colIndex, header := range belowHeaders {
		colName, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return "", nil, fmt.Errorf("获取列名失败: %v", err)
		}
		col := colName + fmt.Sprintf("%d", startBelowRow)
		if err := f.SetCellValue("CPU资源使用率低于阈值", col, header); err != nil {
			return "", nil, fmt.Errorf("设置表头失败: %v", err)
		}
	}

	// 插入BelowWeekExcelData数据
	for i, item := range data.BelowWeekExcelData {
		row := startBelowRow + i + 1
		rowData := []interface{}{
			item.SystemName,
			item.ApplicationName,
			item.GroupName,
			item.ContainerCpuAvg,
			item.PodCpuAvg,
			item.ContainerMemAvg,
			item.PodMemAvg,
			item.Recommend,
		}
		for colIndex, cellData := range rowData {
			colName, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return "", nil, fmt.Errorf("获取列名失败: %v", err)
			}
			col := colName + fmt.Sprintf("%d", row)
			if err := f.SetCellValue("CPU资源使用率低于阈值", col, cellData); err != nil {
				return "", nil, fmt.Errorf("写入数据失败: %v", err)
			}
		}
	}

	// 3. 生成临时文件
	tempFile, err := os.CreateTemp("", "finops_*.xlsx")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// 4. 验证生成
	if _, err := f.WriteTo(tempFile); err != nil {
		return "", nil, fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 5. 读取文件内容
	fileBytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return "FinOps_系统资源使用分析数据.xlsx", fileBytes, nil
}

// ProcessExcelTemplate 处理Excel模板并插入数据
func ProcessExcelTemplate(templatePath string, data MailTotalDataInfo) ([]byte, error) {
	// 打开Excel模板
	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("打开Excel模板失败: %v", err)
	}
	defer f.Close()

	// 设置报告时间范围
	if err := f.SetCellValue("Sheet1", "A1", fmt.Sprintf("报告时间: %s 至 %s", data.StartTime, data.EndTime)); err != nil {
		return nil, fmt.Errorf("设置报告时间失败: %v", err)
	}

	// 处理OverWeekData（资源使用率较高的数据）
	startRow := 3 // 从第3行开始插入数据
	if err := f.SetCellValue("Sheet1", "A2", "CPU资源使用率超过阈值"); err != nil {
		return nil, fmt.Errorf("设置标题失败: %v", err)
	}
	// 设置表头
	headers := []string{"系统名称", "应用名称", "部署组名", "CPU Container(%)", "CPU Pod(%)", "内存 Container(%)", "内存 Pod(%)"}
	for colIndex, header := range headers {
		colName, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return nil, fmt.Errorf("获取列名失败: %v", err)
		}
		col := colName + "3"
		if err := f.SetCellValue("Sheet1", col, header); err != nil {
			return nil, fmt.Errorf("设置表头失败: %v", err)
		}
	}

	// 插入OverWeekData数据
	for i, item := range data.OverWeekData {
		row := startRow + i + 1
		rowData := []interface{}{
			item.ApplicationName,
			item.SystemName,
			item.GroupName,
			item.ContainerCpuAvg,
			item.PodCpuAvg,
			item.ContainerMemAvg,
			item.PodMemAvg,
		}
		for colIndex, cellData := range rowData {
			colName, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return nil, fmt.Errorf("获取列名失败: %v", err)
			}
			col := colName + fmt.Sprintf("%d", row)
			if err := f.SetCellValue("Sheet1", col, cellData); err != nil {
				return nil, fmt.Errorf("写入数据失败: %v", err)
			}
		}
	}

	// 处理BelowWeekData（资源使用率较低的数据）
	startBelowRow := startRow + len(data.OverWeekData) + 3 // 在OverWeekData下方留出3行空白
	if err := f.SetCellValue("Sheet1", "A"+fmt.Sprintf("%d", startBelowRow-1), "资源使用率低于阈值"); err != nil {
		return nil, fmt.Errorf("设置标题失败: %v", err)
	}
	// 设置BelowWeekData表头
	belowHeaders := []string{"系统名称", "应用名称", "组名", "Pod ID", "容器CPU平均(%)", "Pod CPU平均(%)"}
	for colIndex, header := range belowHeaders {
		colName, err := excelize.ColumnNumberToName(colIndex + 1)
		if err != nil {
			return nil, fmt.Errorf("获取列名失败: %v", err)
		}
		col := colName + fmt.Sprintf("%d", startBelowRow)
		if err := f.SetCellValue("Sheet1", col, header); err != nil {
			return nil, fmt.Errorf("设置表头失败: %v", err)
		}
	}

	// 插入BelowWeekData数据
	for i, item := range data.BelowWeekData {
		row := startBelowRow + i + 1
		rowData := []interface{}{
			item.ApplicationName,
			item.SystemName,
			item.GroupName,
			item.ContainerCpuAvg,
			item.PodCpuAvg,
		}
		for colIndex, cellData := range rowData {
			colName, err := excelize.ColumnNumberToName(colIndex + 1)
			if err != nil {
				return nil, fmt.Errorf("获取列名失败: %v", err)
			}
			col := colName + fmt.Sprintf("%d", row)
			if err := f.SetCellValue("Sheet1", col, cellData); err != nil {
				return nil, fmt.Errorf("写入数据失败: %v", err)
			}
		}
	}

	// 保存修改后的Excel到临时缓冲区
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("保存Excel文件失败: %v", err)
	}

	return buf.Bytes(), nil
}

// ConstructAttachment1 创建附件
func ConstructAttachment1(filePath string) (string, []byte, error) {
	// 读取文件内容
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("读取文件失败: %v", err)
	}

	// 提取文件名
	fileName := filepath.Base(filePath)

	// 处理失败，回退到直接读取文件
	// 将文件内容转换为base64
	encodedFile := make([]byte, base64.StdEncoding.EncodedLen(len(fileContent)))
	base64.StdEncoding.Encode(encodedFile, fileContent)

	return fileName, encodedFile, nil
}

// CreateExcelAttachmentWithData1 创建带数据的Excel附件
func CreateExcelAttachmentWithData1(templatePath string, data MailTotalDataInfo) (string, []byte, error) {
	// 处理Excel模板
	processedFile, err := ProcessExcelTemplate(templatePath, data)
	if err != nil {
		return "", nil, fmt.Errorf("处理Excel模板失败: %v", err)
	}

	// 提取文件名
	fileName := filepath.Base(templatePath)

	// 将处理后的文件内容转换为base64
	encodedFile := make([]byte, base64.StdEncoding.EncodedLen(len(processedFile)))
	base64.StdEncoding.Encode(encodedFile, processedFile)

	return fileName, encodedFile, nil
}
