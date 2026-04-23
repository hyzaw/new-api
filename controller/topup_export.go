package controller

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const adminTopUpExportLimit = 50000

type xlsxCell struct {
	Value string
}

func ExportAllTopUps(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	topUps, err := model.GetAdminTopUpsForExport(keyword, adminTopUpExportLimit)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items, err := model.BuildAdminTopUpItems(topUps)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	content, err := buildTopUpOrdersXLSX(items)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	filename := fmt.Sprintf("topup-orders-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", filename, url.QueryEscape(filename)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", content)
}

func buildTopUpOrdersXLSX(items []*model.AdminTopUpItem) ([]byte, error) {
	headers := []string{
		"订单ID",
		"用户ID",
		"用户名",
		"订单号",
		"支付方式",
		"充值数量",
		"到账额度",
		"支付金额",
		"订单状态",
		"退款状态",
		"退款笔数",
		"已申请退款",
		"已成功退款",
		"待确认退款",
		"可退款",
		"邀请返利用户ID",
		"邀请返利额度",
		"已回退邀请返利",
		"邀请返利时间",
		"创建时间",
		"完成时间",
	}

	rows := make([][]xlsxCell, 0, len(items)+1)
	headerRow := make([]xlsxCell, 0, len(headers))
	for _, header := range headers {
		headerRow = append(headerRow, xlsxCell{Value: header})
	}
	rows = append(rows, headerRow)

	for _, item := range items {
		if item == nil {
			continue
		}
		rows = append(rows, []xlsxCell{
			{Value: strconv.Itoa(item.Id)},
			{Value: strconv.Itoa(item.UserId)},
			{Value: item.Username},
			{Value: item.TradeNo},
			{Value: topUpPaymentMethodLabel(item.PaymentMethod)},
			{Value: strconv.FormatInt(item.Amount, 10)},
			{Value: strconv.Itoa(item.GrantedQuota)},
			{Value: formatExportMoney(item.Money)},
			{Value: topUpStatusLabel(item.Status)},
			{Value: topUpRefundStatusLabel(item.RefundStatus)},
			{Value: strconv.Itoa(item.RefundCount)},
			{Value: formatExportMoney(item.RequestedRefundAmount)},
			{Value: formatExportMoney(item.SuccessfulRefundAmount)},
			{Value: formatExportMoney(item.PendingRefundAmount)},
			{Value: formatExportMoney(item.RefundableAmount)},
			{Value: formatOptionalInt(item.InviteRebateUserId)},
			{Value: strconv.Itoa(item.InviteRebateQuota)},
			{Value: strconv.Itoa(item.InviteRebateRefundedQuota)},
			{Value: formatExportTimestamp(item.InviteRebateTime)},
			{Value: formatExportTimestamp(item.CreateTime)},
			{Value: formatExportTimestamp(item.CompleteTime)},
		})
	}

	return buildSimpleXLSX("充值订单", rows)
}

func buildSimpleXLSX(sheetName string, rows [][]xlsxCell) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	files := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets><sheet name="` + escapeXMLAttr(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>` +
			`</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>` +
			`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
			`</Relationships>`,
		"xl/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
			`<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
			`<fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts>` +
			`<fills count="1"><fill><patternFill patternType="none"/></fill></fills>` +
			`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
			`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
			`<cellXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/></cellXfs>` +
			`</styleSheet>`,
		"xl/worksheets/sheet1.xml": buildWorksheetXML(rows),
	}

	order := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"xl/workbook.xml",
		"xl/_rels/workbook.xml.rels",
		"xl/styles.xml",
		"xl/worksheets/sheet1.xml",
	}
	for _, name := range order {
		if err := writeZipFile(zw, name, files[name]); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildWorksheetXML(rows [][]xlsxCell) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	b.WriteString(`<sheetData>`)
	for rowIndex, row := range rows {
		b.WriteString(`<row r="`)
		b.WriteString(strconv.Itoa(rowIndex + 1))
		b.WriteString(`">`)
		for columnIndex, cell := range row {
			ref := columnName(columnIndex+1) + strconv.Itoa(rowIndex+1)
			b.WriteString(`<c r="`)
			b.WriteString(ref)
			b.WriteString(`" t="inlineStr"><is><t>`)
			b.WriteString(escapeXMLText(cell.Value))
			b.WriteString(`</t></is></c>`)
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func writeZipFile(zw *zip.Writer, name string, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(content))
	return err
}

func columnName(index int) string {
	var name []byte
	for index > 0 {
		index--
		name = append([]byte{byte('A' + index%26)}, name...)
		index /= 26
	}
	return string(name)
}

func escapeXMLText(value string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

func escapeXMLAttr(value string) string {
	return strings.ReplaceAll(escapeXMLText(value), `"`, "&quot;")
}

func formatExportMoney(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func formatOptionalInt(value int) string {
	if value == 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func formatExportTimestamp(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
}

func topUpPaymentMethodLabel(value string) string {
	switch value {
	case model.PaymentMethodStripe:
		return "Stripe"
	case model.PaymentMethodCreem:
		return "Creem"
	case model.PaymentMethodWaffo:
		return "Waffo"
	case model.PaymentMethodWaffoPancake:
		return "Waffo Pancake"
	case "alipay":
		return "支付宝"
	case "alipay_f2f":
		return "支付宝当面付"
	case "wxpay":
		return "微信"
	default:
		return value
	}
}

func topUpStatusLabel(value string) string {
	switch value {
	case common.TopUpStatusSuccess:
		return "成功"
	case common.TopUpStatusPending:
		return "待支付"
	case common.TopUpStatusFailed:
		return "失败"
	case common.TopUpStatusExpired:
		return "已关闭"
	default:
		return value
	}
}

func topUpRefundStatusLabel(value string) string {
	switch value {
	case "none":
		return "未退款"
	case "partial":
		return "部分退款"
	case "pending":
		return "退款处理中"
	case "full":
		return "已全额退款"
	default:
		return value
	}
}
