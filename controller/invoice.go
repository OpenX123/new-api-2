package controller

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type createInvoiceRequest struct {
	OrderKeys []struct {
		Type string `json:"type"`
		Id   int    `json:"id"`
	} `json:"order_keys"`
	TitleType     int    `json:"title_type"`
	TitleName     string `json:"title_name"`
	TaxNo         string `json:"tax_no"`
	Email         string `json:"email"`
	Remark        string `json:"remark"`
	SaveAsDefault bool   `json:"save_as_default"`
}

// GetInvoiceableOrders GET /api/user/invoice/orders —— 同时返回默认抬头
func GetInvoiceableOrders(c *gin.Context) {
	userId := c.GetInt("id")
	orders, err := model.GetInvoiceableOrders(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var title *dto.InvoiceTitle
	if user, err := model.GetUserById(userId, false); err == nil {
		setting := user.GetSetting()
		title = setting.InvoiceTitle
	}
	common.ApiSuccess(c, gin.H{"orders": orders, "default_title": title})
}

// GetUserInvoiceList GET /api/user/invoice/self
func GetUserInvoiceList(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	list, total, err := model.GetUserInvoices(userId, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(list)
	common.ApiSuccess(c, pageInfo)
}

// CreateInvoice POST /api/user/invoice
func CreateInvoice(c *gin.Context) {
	userId := c.GetInt("id")
	var req createInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.OrderKeys) == 0 {
		common.ApiErrorMsg(c, "请选择要开票的订单")
		return
	}
	if req.TitleType != model.InvoiceTitleTypePersonal && req.TitleType != model.InvoiceTitleTypeCompany {
		common.ApiErrorMsg(c, "抬头类型无效")
		return
	}
	if req.TitleName == "" || req.Email == "" {
		common.ApiErrorMsg(c, "抬头名称与接收邮箱不能为空")
		return
	}
	if req.TitleType == model.InvoiceTitleTypeCompany && req.TaxNo == "" {
		common.ApiErrorMsg(c, "企业抬头必须填写税号")
		return
	}
	// 服务端核验：仅允许选择当前可开票订单（归属本人/success/未占用），占用竞态由唯一索引兜底
	available, err := model.GetInvoiceableOrders(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	availMap := make(map[string]*model.InvoiceableOrder, len(available))
	for _, o := range available {
		availMap[o.OrderType+"#"+strconv.Itoa(o.OrderId)] = o
	}
	orders := make([]*model.InvoiceOrder, 0, len(req.OrderKeys))
	for _, k := range req.OrderKeys {
		o, ok := availMap[k.Type+"#"+strconv.Itoa(k.Id)]
		if !ok {
			common.ApiErrorMsg(c, fmt.Sprintf("订单不可开票或不存在: %s#%d", k.Type, k.Id))
			return
		}
		orders = append(orders, &model.InvoiceOrder{OrderType: o.OrderType, OrderId: o.OrderId,
			TradeNo: o.TradeNo, Money: o.Money})
	}
	inv := &model.Invoice{UserId: userId, TitleType: req.TitleType, TitleName: req.TitleName,
		TaxNo: req.TaxNo, Email: req.Email, Remark: req.Remark}
	if err := model.CreateInvoiceWithOrders(inv, orders); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.SaveAsDefault {
		saveDefaultInvoiceTitle(userId, &req)
	}
	common.ApiSuccess(c, inv)
}

func saveDefaultInvoiceTitle(userId int, req *createInvoiceRequest) {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		common.SysError("failed to load user for invoice title: " + err.Error())
		return
	}
	setting := user.GetSetting()
	setting.InvoiceTitle = &dto.InvoiceTitle{TitleType: req.TitleType,
		TitleName: req.TitleName, TaxNo: req.TaxNo, Email: req.Email}
	user.SetSetting(setting)
	if err := user.Update(false); err != nil {
		common.SysError("failed to save default invoice title: " + err.Error())
	}
}

// CancelInvoice DELETE /api/user/invoice/:id
func CancelInvoice(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.CancelInvoice(id, userId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// DownloadInvoiceFileUser GET /api/user/invoice/:id/file
func DownloadInvoiceFileUser(c *gin.Context) {
	userId := c.GetInt("id")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	inv, err := model.GetInvoiceById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if inv.UserId != userId {
		common.ApiErrorMsg(c, "无权访问该发票")
		return
	}
	serveInvoiceFile(c, id)
}

func serveInvoiceFile(c *gin.Context, invoiceId int) {
	f, err := model.GetInvoiceFile(invoiceId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	safeFilename := strings.NewReplacer("\"", "", "\r", "", "\n", "").Replace(f.Filename)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		safeFilename, url.PathEscape(safeFilename)))
	c.Data(200, f.MimeType, f.Data)
}

var invoiceAllowedMime = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
}

// GetAllInvoices GET /api/invoice/
func GetAllInvoices(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	status, _ := strconv.Atoi(c.Query("status")) // 0=全部
	list, total, err := model.GetAllInvoices(keyword, status, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(list)
	common.ApiSuccess(c, pageInfo)
}

// GetInvoice GET /api/invoice/:id
func GetInvoice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	inv, err := model.GetInvoiceById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, inv)
}

// UploadInvoiceFile POST /api/invoice/:id/file  (multipart field: file)
func UploadInvoiceFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请选择发票文件")
		return
	}
	if fh.Size <= 0 || fh.Size > model.InvoiceFileMaxSize {
		common.ApiErrorMsg(c, "文件大小须在 10MB 以内")
		return
	}
	src, err := fh.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, model.InvoiceFileMaxSize+1))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(data) > model.InvoiceFileMaxSize {
		common.ApiErrorMsg(c, "文件大小须在 10MB 以内")
		return
	}
	mimeType := http.DetectContentType(data)
	if !invoiceAllowedMime[mimeType] {
		common.ApiErrorMsg(c, "仅支持 PDF / PNG / JPG 格式")
		return
	}
	if err := model.CompleteInvoiceWithFile(id, fh.Filename, mimeType, data); err != nil {
		common.ApiError(c, err)
		return
	}
	if inv, err := model.GetInvoiceById(id); err == nil {
		go sendInvoiceCompletedEmail(inv, fh.Filename, mimeType, data)
	}
	common.ApiSuccess(c, nil)
}

// RejectInvoiceAdmin POST /api/invoice/:id/reject  {reason}
func RejectInvoiceAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		common.ApiErrorMsg(c, "驳回原因不能为空")
		return
	}
	inv, err := model.GetInvoiceById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.RejectInvoice(id, req.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	go sendInvoiceRejectedEmail(inv, req.Reason)
	common.ApiSuccess(c, nil)
}

// DownloadInvoiceFileAdmin GET /api/invoice/:id/file
func DownloadInvoiceFileAdmin(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	serveInvoiceFile(c, id)
}

func sendInvoiceCompletedEmail(inv *model.Invoice, filename, mimeType string, data []byte) {
	subject := fmt.Sprintf("%s - 发票已开具 (%s)", common.SystemName, inv.InvoiceNo)
	content := fmt.Sprintf("<p>您好，</p><p>您的开票申请 <b>%s</b>（抬头：%s，金额：%.2f）已开具完成，发票文件见附件。</p><p>也可登录控制台在 钱包 → 发票 中下载。</p>",
		inv.InvoiceNo, html.EscapeString(inv.TitleName), inv.Money)
	if err := common.SendEmailWithAttachment(subject, inv.Email, content, filename, mimeType, data); err != nil {
		common.SysError("failed to send invoice email: " + err.Error())
	}
}

func sendInvoiceRejectedEmail(inv *model.Invoice, reason string) {
	subject := fmt.Sprintf("%s - 开票申请已驳回 (%s)", common.SystemName, inv.InvoiceNo)
	content := fmt.Sprintf("<p>您好，</p><p>您的开票申请 <b>%s</b>（抬头：%s，金额：%.2f）已被驳回。</p><p>原因：%s</p><p>相关订单已释放，您可修改信息后重新申请。</p>",
		inv.InvoiceNo, html.EscapeString(inv.TitleName), inv.Money, html.EscapeString(reason))
	if err := common.SendEmail(subject, inv.Email, content); err != nil {
		common.SysError("failed to send invoice reject email: " + err.Error())
	}
}
