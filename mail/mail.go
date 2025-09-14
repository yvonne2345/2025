package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"

	"fmt"
	"github.com/xuri/excelize/v2"
	"html/template"
	"io/ioutil"
	"log"
	"mime"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MailServerConfig 邮件服务器配置
type MailServerConfig struct {
	SMTPServer string
	SMTPPort   int
	User       string
	Password   string
	Alias      string
}

/*use unSSL to link mail server*/
type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true
	_, resp, th := a.Auth.Start(&s)
	return "LOGIN", resp, th
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	command := string(fromServer)
	command = strings.TrimSpace(command)
	command = strings.TrimSuffix(command, ":")
	command = strings.ToLower(command)
	if more {
		if command == "username" {
			return []byte(fmt.Sprintf("%s", a.username)), nil
		} else if command == "password" {
			return []byte(fmt.Sprintf("%s", a.password)), nil
		} else {
			// We've already sent everything.
			return nil, fmt.Errorf("unexpected server challenge: %s", command)
		}
	}
	return nil, nil
}

// constructMIMEImage 创建正文中插入的图片
func constructMIMEImage(cid, imagePath string) (string, []byte, error) {
	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		log.Println(fmt.Sprintf("正文嵌入内容：%s 打开失败：%v", imagePath, err))
		return "", nil, err
	}

	// 获取图片MIME类型
	ext := filepath.Ext(imagePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// 编码图片为base64
	encodedImage := base64.StdEncoding.EncodeToString(imageData)

	return mimeType, []byte(encodedImage), nil
}

// encodeSubject 编码邮件主题
func encodeSubject(subject string) string {
	// 简单的base64编码实现
	data := []byte(subject)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	return fmt.Sprintf("=?UTF-8?B?%s?=", encoded)
}

// encodeFileName 编码文件名
func encodeFileName(filename string) string {
	// 简单的base64编码实现
	data := []byte(filename)
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)
	return fmt.Sprintf("=?UTF-8?B?%s?=", encoded)
}

func DailySendEmail() {
	// 测试验证数据
	toReceiverList := []string{"1", "2"}
	ccReceiverList := []string{"1", "2"}
	mailTitle := "FinOps系统资源使用分析报告"
	// 定义模板文件路径
	fmt.Println("==========")
	log.Println("==========")

	//构建报告表格数据
	//TODO cpu、内存使用量
	info := MailTotalDataInfo{}

	htmlBody := renderHTML("template/finops_table_new.html", info)

	// 构建Excel附件|构建excel数据
	//TODO cpu、内存使用量
	excelInfo := MailExcelDataInfo{}
	templatePath := "template/FinOps.xlsx"
	fileName, encodedFile, err := CreateExcelAttachmentWithData(templatePath, excelInfo)
	if err != nil {
		fmt.Println("构建Excel附件失败:", err)
		// 回退到普通附件
		fileName, encodedFile, err = ConstructAttachment(templatePath)
		if err != nil {
			fmt.Println("回退失败:", err)
		}
	}
	if err == nil {
		fmt.Println("构建Excel附件失败")
	}

	SendEmail(toReceiverList, ccReceiverList, mailTitle, fileName, encodedFile, nil, nil, nil, htmlBody)
}

func SendEmail(toReceiverList, ccReceiverList []string, mailTitle, fileName string, encodedFile []byte, imageDict map[string][2]string, fileList []string, mailServer *MailServerConfig, htmlBody string) bool {
	// 设置邮件服务器配置
	var (
		smtpServer string
		smtpPort   int
		user       string
		passwd     string
		alias      string
	)

	if mailServer != nil {
		smtpServer = mailServer.SMTPServer
		smtpPort = mailServer.SMTPPort
		user = mailServer.User
		passwd = mailServer.Password
		alias = mailServer.Alias
	} else {
		// 默认配置
		smtpServer = "21.0.0.76"
		smtpPort = 25
		user = "RPA@cpic.com.cn"
		passwd = "SXzdh@YsL719"
		alias = "寿险运维自动化"
	}

	// 构建邮件头部
	from := mail.Address{Name: alias, Address: user}
	to := make([]string, 0)
	receiverList := append(toReceiverList, ccReceiverList...)
	for _, addr := range receiverList {
		to = append(to, addr)
	}

	// 构建完整邮件体
	boundary := "----" + hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	buffer := new(bytes.Buffer)

	// 邮件头部
	buffer.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
	buffer.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toReceiverList, "; ")))
	if len(ccReceiverList) > 0 {
		buffer.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccReceiverList, "; ")))
	}
	buffer.WriteString(fmt.Sprintf("Subject: %s\r\n", encodeSubject(mailTitle)))
	buffer.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=%s\r\n\r\n", boundary))

	// HTML正文部分
	buffer.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buffer.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
	buffer.WriteString(htmlBody)

	// 附件excel部分
	buffer.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buffer.WriteString(fmt.Sprintf("Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet; charset=UTF-8; name=%q\r\n", fileName))
	buffer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%q\r\n", fileName))
	buffer.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	// 对文件内容进行base64编码后再写入
	encodedData := make([]byte, base64.StdEncoding.EncodedLen(len(encodedFile)))
	base64.StdEncoding.Encode(encodedData, encodedFile)
	buffer.Write(encodedData)

	buffer.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	// 发送邮件
	auth := LoginAuth(user, passwd)

	smtpAddr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)

	err := smtp.SendMail(smtpAddr, auth, user, to, buffer.Bytes())
	if err != nil {
		fmt.Println(fmt.Sprintf("邮件发送失败：%v", err))
		return false
	}

	fmt.Println("邮件发送成功")
	return true
}

func renderHTML(tplPath string, data MailTotalDataInfo) string {
	tmpl := template.Must(template.ParseFiles(tplPath))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

// ConstructAttachment 创建邮件附件
func ConstructAttachment(filePath string) (string, []byte, error) {
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", nil, err
	}

	// 获取文件名
	fileName := filepath.Base(filePath)

	// 编码文件为base64
	encodedFile := base64.StdEncoding.EncodeToString(fileData)

	return fileName, []byte(encodedFile), nil
}

func CreateExcelAttachmentWithData(templatePath string, data MailExcelDataInfo) (string, []byte, error) {
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

// 处理mailContext
//func GenerateData() MailTotalDataInfo {
//	//cpu和mem,或
//	//cpu pod>0.4,container>0.6
//	//mem container>0.95
//	var info MailTotalDataInfo
//	var CpuLimitOverWeekData []model.CpuLimitWeekData
//	info.EndTime = time.Now().Add(-24 * time.Hour).Format("2006-01-02")
//	info.StartTime = time.Now().Add(-24 * 7 * time.Hour).Format("2006-01-02")
//	err := AUTOMATED_DB.Table("container_cpu_limit_week_values as c").
//		Select("c.group_name as group_name, round(avg(c.average*100),1) as container_avg, round(avg(p.average*100),1) as pod_avg").
//		Joins("JOIN pod_cpu_limit_week_values as p ON c.group_name = p.group_name").
//		Where("p.average >= ? AND c.average >= ? AND c.creat_time >= ?", 0.4, 0.6, time.Now().Format("2006-01-02")).
//		Group("c.group_name").
//		Order("c.average desc").
//		Limit(10).Scan(&CpuLimitOverWeekData)
//	if err != nil {
//		log.Println("err", err)
//	}
//	log.Println("1、CpuLimitOverWeekData ", CpuLimitOverWeekData)
//
//	// 2、初始化OverWeekData切片，长度与CpuOverWeekData相同
//	OverWeekData := make([]model.OverWeekData, len(CpuLimitOverWeekData))
//	for i, data := range CpuLimitOverWeekData {
//		//根据group_name查application_name和cimsid
//		var MailAppInfo model.MailAppInfo
//		//log.Println("2、group_name", data.GroupName)
//
//		//err1 := AUTOMATED_DB.Table("app_groups ag").Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name").Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo).Error
//		//var recommend string
//		//AUTOMATED_DB.Table("agent_recommend_weeks").Select("recommend").Where("group_name =? and creat_time >=?", data.GroupName, time.Now().Format("2006-01-02")).Scan(&recommend)
//
//		err1 := AUTOMATED_DB.Table("app_groups ag").
//			Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").
//			Joins("LEFT JOIN agent_recommend_weeks arw ON ag.group_name = arw.group_name AND arw.creat_time >= ?", time.Now().Format("2006-01-02")).
//			Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name,arw.recommend as recommend").
//			Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo).Error
//		fmt.Println("---MailAppInfo---", MailAppInfo)
//
//		//TODO 3、mem数据有问题 >0.95
//		var MemLimitWeekData model.MemLimitWeekData
//		AUTOMATED_DB.Table("container_memory_limit_week_values c").
//			Select("c.group_name as group_name, ROUND (avg(c.average * 100), 1)as container_avg, ROUND (avg(p.average * 100), 1)as pod_avg").
//			Joins("JOIN pod_memory_limit_week_values p ON c.group_name = p.group_name").
//			Where("c.group_name=? and c.creat_time >= ?", data.GroupName, time.Now().Format("2006-01-02")).
//			Group("c.group_name").
//			Order("c.average desc").
//			Limit(10).Scan(&MemLimitWeekData)
//		if err1 != nil {
//			log.Println("err1", err1)
//		}
//		OverWeekData[i].ContainerCpuAvg = data.ContainerAvg
//		OverWeekData[i].PodCpuAvg = data.PodAvg
//		OverWeekData[i].ApplicationName = MailAppInfo.ApplicationName
//		OverWeekData[i].SystemName = MailAppInfo.SystemName
//		OverWeekData[i].GroupName = MailAppInfo.GroupName
//		OverWeekData[i].PodMemAvg = MemLimitWeekData.PodAvg
//		OverWeekData[i].ContainerMemAvg = MemLimitWeekData.ContainerAvg
//		OverWeekData[i].Recommend = MailAppInfo.Recommend
//	}
//	info.OverWeekData = OverWeekData
//	//log.Println("2、OverWeekData", OverWeekData)
//
//	//pod<0.15,container<0.3
//	var CpuLimitBelowWeekData []model.CpuLimitWeekData
//	AUTOMATED_DB.Table("container_cpu_limit_week_values as c").
//		Select("c.group_name as group_name, round(avg(c.average*100),1) as container_avg, round(avg(p.average*100),1) as pod_avg").
//		Joins("JOIN pod_cpu_limit_week_values as p ON c.group_name = p.group_name").
//		Where("p.average >= ? AND p.average < ? AND c.average >= ? AND c.average < ? AND c.creat_time >= ?", 0.001, 0.15, 0.001, 0.3, time.Now().Format("2006-01-02")).
//		Group("c.group_name").
//		Order("c.average asc").
//		Limit(10).Scan(&CpuLimitBelowWeekData)
//
//	if err != nil {
//		log.Println("err", err)
//	}
//	//log.Print("3、CpuLimitBelowWeekData", CpuLimitBelowWeekData)
//	// 4、初始化BelowWeekData切片，长度与CpuBelowWeekData相同
//	BelowWeekData := make([]model.BelowWeekData, len(CpuLimitBelowWeekData))
//	for i, data := range CpuLimitBelowWeekData {
//		//根据group_name查application_name和systemname
//		var MailAppInfo model.MailAppInfo
//		AUTOMATED_DB.Table("app_groups ag").
//			Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").
//			Joins("LEFT JOIN agent_recommend_weeks arw ON ag.group_name = arw.group_name AND arw.creat_time >= ?", time.Now().Format("2006-01-02")).
//			Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name,arw.recommend as recommend").
//			Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo)
//		//fmt.Println("---MailAppInfo---", MailAppInfo)
//
//		var MemLimitWeekData model.MemLimitWeekData
//		AUTOMATED_DB.Table("container_memory_limit_week_values c").
//			Select("c.group_name as group_name, ROUND (avg(c.average * 100), 1)as container_avg, ROUND (avg(p.average * 100), 1)as pod_avg").
//			Joins("JOIN pod_memory_limit_week_values p ON c.group_name = p.group_name").
//			Where("c.group_name=? and c.creat_time >= ?", data.GroupName, time.Now().Format("2006-01-02")).
//			Group("c.group_name").
//			Order("c.average desc").
//			Limit(10).Scan(&MemLimitWeekData)
//		if err != nil {
//			log.Println("err1", err)
//		}
//		BelowWeekData[i].ContainerCpuAvg = data.ContainerAvg
//		BelowWeekData[i].PodCpuAvg = data.PodAvg
//		BelowWeekData[i].ContainerMemAvg = MemLimitWeekData.ContainerAvg
//		BelowWeekData[i].PodMemAvg = MemLimitWeekData.PodAvg
//		BelowWeekData[i].ApplicationName = MailAppInfo.ApplicationName
//		BelowWeekData[i].SystemName = MailAppInfo.SystemName
//		BelowWeekData[i].GroupName = MailAppInfo.GroupName
//		BelowWeekData[i].Recommend = MailAppInfo.Recommend
//	}
//	info.BelowWeekData = BelowWeekData
//	//log.Print("4、BelowWeekData", BelowWeekData)
//
//	//log.Println("5、info", info)
//
//	return info
//}

// 处理ExcelData
//func GenerateExcelData() model.MailExcelDataInfo {
//	var info model.MailExcelDataInfo
//	var CpuOverWeekData []model.CpuLimitWeekData
//	//log.Println("查询1、CpuOverWeekData")
//	err := AUTOMATED_DB.Table("container_cpu_limit_week_values as c").
//		Select("c.group_name as group_name, round(avg(c.average*100),1) as container_avg, round(avg(p.average*100),1) as pod_avg").
//		Joins("JOIN pod_cpu_limit_week_values as p ON c.group_name = p.group_name").
//		Where("p.average >= ? AND c.average >= ? AND c.creat_time >= ?", 0.4, 0.6, time.Now().Format("2006-01-02")).
//		Group("c.group_name").
//		Order("c.average desc").
//		Scan(&CpuOverWeekData)
//	if err != nil {
//		log.Println("err", err)
//	}
//	//fmt.Println("1、CpuOverWeekData ", CpuOverWeekData)
//	// 2、初始化OverWeekExcelData切片，长度与CpuOverWeekData相同
//	OverWeekExcelData := make([]model.OverWeekExcelData, len(CpuOverWeekData))
//	for i, data := range CpuOverWeekData {
//		//根据group_name查application_name和cimsid
//		var MailAppInfo model.MailAppInfo
//		log.Println("2、group_name", data.GroupName)
//		err1 := AUTOMATED_DB.Table("app_groups ag").
//			Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").
//			Joins("LEFT JOIN agent_recommend_weeks arw ON ag.group_name = arw.group_name AND arw.creat_time >= ?", time.Now().Format("2006-01-02")).
//			Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name,arw.recommend as recommend").
//			Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo).Error
//		//fmt.Println("---MailAppInfo---", MailAppInfo)
//
//		var MemWeekData model.MemLimitWeekData
//		AUTOMATED_DB.Table("container_memory_limit_week_values c").
//			Select("c.group_name as group_name, ROUND (avg(c.average * 100), 1)as container_avg, ROUND (avg(p.average * 100), 1)as pod_avg").
//			Joins("JOIN pod_memory_limit_week_values p ON c.group_name = p.group_name").
//			Where("c.group_name=? and c.creat_time >= ?", data.GroupName, time.Now().Format("2006-01-02")).
//			Group("c.group_name").
//			Order("c.average desc").
//			Scan(&MemWeekData)
//		if err1 != nil {
//			log.Println("err1", err1)
//		}
//		OverWeekExcelData[i].ContainerCpuAvg = data.ContainerAvg
//		OverWeekExcelData[i].PodCpuAvg = data.PodAvg
//		OverWeekExcelData[i].ApplicationName = MailAppInfo.ApplicationName
//		OverWeekExcelData[i].SystemName = MailAppInfo.SystemName
//		OverWeekExcelData[i].GroupName = MailAppInfo.GroupName
//		OverWeekExcelData[i].PodMemAvg = MemWeekData.PodAvg
//		OverWeekExcelData[i].ContainerMemAvg = MemWeekData.ContainerAvg
//		OverWeekExcelData[i].Recommend = MailAppInfo.Recommend
//	}
//
//	//TODO 使用量修改
//	var CpuCoreWeekData []model.CpuCoreWeekData
//	//log.Println("查询1、CpuOverWeekData")
//	err = AUTOMATED_DB.Table("container_cpu_core_week_values as c").
//		Select("c.group_name as group_name, round(avg(c.average*100),1) as container_avg, round(avg(p.average*100),1) as pod_avg").
//		Joins("JOIN pod_cpu_core_week_values as p ON c.group_name = p.group_name").
//		Where("p.average >= ? AND c.average >= ? AND c.creat_time >= ?", 0.4, 0.6, time.Now().Format("2006-01-02")).
//		Group("c.group_name").
//		Order("c.average desc").
//		Scan(&CpuCoreWeekData)
//	if err != nil {
//		log.Println("err", err)
//	}
//	for i, data := range CpuCoreWeekData {
//		//根据group_name查application_name和cimsid
//		var MailAppInfo model.MailAppInfo
//		log.Println("2、group_name", data.GroupName)
//		err1 := AUTOMATED_DB.Table("app_groups ag").
//			Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").
//			Joins("LEFT JOIN agent_recommend_weeks arw ON ag.group_name = arw.group_name AND arw.creat_time >= ?", time.Now().Format("2006-01-02")).
//			Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name,arw.recommend as recommend").
//			Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo).Error
//		//fmt.Println("---MailAppInfo---", MailAppInfo)
//
//		var MemWeekData model.MemLimitWeekData
//		AUTOMATED_DB.Table("container_memory_limit_week_values c").
//			Select("c.group_name as group_name, ROUND (avg(c.average * 100), 1)as container_avg, ROUND (avg(p.average * 100), 1)as pod_avg").
//			Joins("JOIN pod_memory_limit_week_values p ON c.group_name = p.group_name").
//			Where("c.group_name=? and c.creat_time >= ?", data.GroupName, time.Now().Format("2006-01-02")).
//			Group("c.group_name").
//			Order("c.average desc").
//			Scan(&MemWeekData)
//		if err1 != nil {
//			log.Println("err1", err1)
//		}
//		OverWeekExcelData[i].ContainerCpuAvg = data.ContainerAvg
//		OverWeekExcelData[i].PodCpuAvg = data.PodAvg
//		OverWeekExcelData[i].ApplicationName = MailAppInfo.ApplicationName
//		OverWeekExcelData[i].SystemName = MailAppInfo.SystemName
//		OverWeekExcelData[i].GroupName = MailAppInfo.GroupName
//		OverWeekExcelData[i].PodMemAvg = MemWeekData.PodAvg
//		OverWeekExcelData[i].ContainerMemAvg = MemWeekData.ContainerAvg
//		OverWeekExcelData[i].Recommend = MailAppInfo.Recommend
//	}
//
//	info.OverWeekExcelData = OverWeekExcelData
//	//log.Println("2、OverWeekData", OverWeekData)
//
//	//pod<0.15,container<0.3
//	var CpuBelowWeekData []model.CpuLimitWeekData
//	//log.Println("查询3、CpuBelowWeekData")
//
//	AUTOMATED_DB.Table("container_cpu_limit_week_values as c").
//		Select("c.group_name as group_name, round(avg(c.average*100),1) as container_avg, round(avg(p.average*100),1) as pod_avg").
//		Joins("JOIN pod_cpu_limit_week_values as p ON c.group_name = p.group_name").
//		Where("p.average >= ? AND p.average < ? AND c.average >= ? AND c.average < ? AND c.creat_time >= ?", 0.001, 0.15, 0.001, 0.3, time.Now().Format("2006-01-02")).
//		Group("c.group_name").
//		Order("c.average asc").
//		Scan(&CpuBelowWeekData)
//
//	if err != nil {
//		log.Println("err", err)
//	}
//
//	//TODO 使用量
//
//	//log.Print("3、CpuBelowWeekData", CpuBelowWeekData)
//	// 4、初始化BelowWeekExcelData切片，长度与CpuBelowWeekData相同
//	BelowWeekExcelData := make([]model.BelowWeekExcelData, len(CpuBelowWeekData))
//	for i, data := range CpuBelowWeekData {
//		//根据group_name查application_name和systemname
//		var MailAppInfo model.MailAppInfo
//		AUTOMATED_DB.Table("app_groups ag").
//			Joins("JOIN cims_systems_info c ON ag.cimsid=c.cimsid").
//			Joins("LEFT JOIN agent_recommend_weeks arw ON ag.group_name = arw.group_name AND arw.creat_time >= ?", time.Now().Format("2006-01-02")).
//			Select("ag.application_name as application_name,ag.group_name as group_name,c.systemname as system_name,arw.recommend as recommend").
//			Where("ag.group_name =?", data.GroupName).Limit(1).Scan(&MailAppInfo)
//		//fmt.Println("---MailAppInfo---", MailAppInfo)
//
//		var MemLimitWeekData model.MemLimitWeekData
//		AUTOMATED_DB.Table("container_memory_limit_week_values c").
//			Select("c.group_name as group_name, ROUND (avg(c.average * 100), 1)as container_avg, ROUND (avg(p.average * 100), 1)as pod_avg").
//			Joins("JOIN pod_memory_limit_week_values p ON c.group_name = p.group_name").
//			Where("c.group_name=? and c.creat_time >= ?", data.GroupName, time.Now().Format("2006-01-02")).
//			Group("c.group_name").
//			Order("c.average desc").
//			Limit(10).Scan(&MemLimitWeekData)
//		if err != nil {
//			log.Println("err1", err)
//		}
//		BelowWeekExcelData[i].ContainerCpuAvg = data.ContainerAvg
//		BelowWeekExcelData[i].PodCpuAvg = data.PodAvg
//		BelowWeekExcelData[i].ContainerMemAvg = MemLimitWeekData.ContainerAvg
//		BelowWeekExcelData[i].PodMemAvg = MemLimitWeekData.PodAvg
//		BelowWeekExcelData[i].ApplicationName = MailAppInfo.ApplicationName
//		BelowWeekExcelData[i].SystemName = MailAppInfo.SystemName
//		BelowWeekExcelData[i].GroupName = MailAppInfo.GroupName
//		BelowWeekExcelData[i].Recommend = MailAppInfo.Recommend
//	}
//	info.BelowWeekExcelData = BelowWeekExcelData
//	//log.Print("4、BelowWeekData", BelowWeekData)
//
//	log.Println("5、info", info)
//
//	return info
//}

func DailySendEmail1() {
	// 测试验证数据
	toReceiverList := []string{"1", "2"}
	ccReceiverList := []string{"1", "2"}
	mailTitle := "FinOps系统资源使用分析报告"
	// 定义模板文件路径
	fmt.Println("==========")
	log.Println("==========")

	var info MailTotalDataInfo
	var week1 = OverWeekData{
		ApplicationName: "finops",
		SystemName:      "gp18ar",
		GroupName:       "ops",
		ContainerCpuAvg: 0.1,
		PodCpuAvg:       0.2,
		ContainerMemAvg: 0.3,
		PodMemAvg:       0.4,
	}
	var week3 = OverWeekData{
		ApplicationName: "finops",
		SystemName:      "gp18ar",
		GroupName:       "ops",
		ContainerCpuAvg: 0.1,
		PodCpuAvg:       0.2,
		ContainerMemAvg: 0.3,
		PodMemAvg:       0.4,
	}
	var week2 = BelowWeekData{
		ApplicationName: "finops2",
		SystemName:      "gp18ar2",
		GroupName:       "ops2",
		ContainerCpuAvg: 0.5,
		PodCpuAvg:       0.6,
	}
	info.OverWeekData = append(info.OverWeekData, week1)
	info.OverWeekData = append(info.OverWeekData, week3)
	info.BelowWeekData = append(info.BelowWeekData, week2)

	htmlBody := renderHTML("template/finops_table_new.html", info)
	// 构建带数据的Excel附件
	templatePath := "template/FinOps.xlsx"

	var info1 MailExcelDataInfo
	var week4 = OverWeekExcelData{
		ApplicationName: "finops",
		SystemName:      "gp18ar",
		GroupName:       "ops",
		ContainerCpuAvg: 0.1,
		PodCpuAvg:       0.2,
		ContainerMemAvg: 0.3,
		PodMemAvg:       0.4,
	}
	var week5 = OverWeekExcelData{
		ApplicationName: "finops",
		SystemName:      "gp18ar",
		GroupName:       "ops",
		ContainerCpuAvg: 0.1,
		PodCpuAvg:       0.2,
		ContainerMemAvg: 0.3,
		PodMemAvg:       0.4,
	}
	var week6 = BelowWeekExcelData{
		ApplicationName: "finops2",
		SystemName:      "gp18ar2",
		GroupName:       "ops2",
		PodId:           "222",
		ContainerCpuAvg: 0.5,
		PodCpuAvg:       0.6,
	}
	info1.OverWeekExcelData = append(info1.OverWeekExcelData, week4)
	info1.OverWeekExcelData = append(info1.OverWeekExcelData, week5)
	info1.BelowWeekExcelData = append(info1.BelowWeekExcelData, week6)

	fileName, encodedFile, err := CreateExcelAttachmentWithData(templatePath, info1)
	fmt.Println("===================")
	if err != nil {
		fmt.Println("构建Excel附件失败:", err)
		// 回退到普通附件
		fileName, encodedFile, err = ConstructAttachment(templatePath)
		if err != nil {
			fmt.Println("回退附件失败:", err)
		}
	}

	SendEmail2(toReceiverList, ccReceiverList, mailTitle, fileName, encodedFile, nil, nil, nil, htmlBody)
}

func SendEmail2(toReceiverList, ccReceiverList []string, mailTitle, fileName string, encodedFile []byte, imageDict map[string][2]string, fileList []string, mailServer *MailServerConfig, htmlBody string) bool {
	// 设置邮件服务器配置
	var (
		smtpServer string
		smtpPort   int
		user       string
		passwd     string
		alias      string
	)

	if mailServer != nil {
		smtpServer = mailServer.SMTPServer
		smtpPort = mailServer.SMTPPort
		user = mailServer.User
		passwd = mailServer.Password
		alias = mailServer.Alias
	} else {
		// 默认配置
		smtpServer = "21.0.0.76"
		smtpPort = 25
		user = "RPA@cpic.com.cn"
		passwd = "SXzdh@YsL719"
		alias = "寿险运维自动化"
	}

	// 构建邮件头部
	from := mail.Address{Name: alias, Address: user}
	to := make([]string, 0)
	receiverList := append(toReceiverList, ccReceiverList...)
	for _, addr := range receiverList {
		to = append(to, addr)
	}

	var buffer bytes.Buffer
	// 邮件头部
	buffer.WriteString(fmt.Sprintf("From: %s\n", from.String()))
	buffer.WriteString(fmt.Sprintf("To: %s\n", strings.Join(toReceiverList, ";")))
	if len(ccReceiverList) > 0 {
		buffer.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(ccReceiverList, ";")))
	}
	buffer.WriteString(fmt.Sprintf("Subject: %s\n", encodeSubject(mailTitle)))
	buffer.WriteString("MIME-Version: 1.0\n")
	buffer.WriteString("Content-Type: text/html; charset=UTF-8\n")
	// 添加一个空行分隔头部和正文
	buffer.WriteString("\n")
	buffer.WriteString(htmlBody)
	//TODO 附件excel
	buffer.WriteString("Content-Type: application/vnd.ms-excel\r\n")
	buffer.WriteString("Content-Transfer-Encoding: base64\r\n")
	buffer.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", fileName))
	buffer.WriteString("\r\n")
	buffer.Write(encodedFile)
	buffer.WriteString("\r\n")

	// 发送邮件
	auth := LoginAuth(user, passwd)

	smtpAddr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)

	err := smtp.SendMail(smtpAddr, auth, user, to, buffer.Bytes())
	if err != nil {
		fmt.Println(fmt.Sprintf("邮件发送失败：%v", err))
		return false
	}

	fmt.Println("邮件发送成功")
	return true
}
