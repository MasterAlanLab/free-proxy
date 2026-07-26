package domain

import "strings"

var countryTranslations = map[string]string{
	"Japan": "日本", "Korea Republic of": "韩国", "Korea": "韩国", "Republic of Korea": "韩国",
	"Thailand": "泰国", "United States": "美国", "United Kingdom": "英国", "Russian Federation": "俄罗斯",
	"Viet Nam": "越南", "Vietnam": "越南", "China": "中国", "Taiwan": "台湾", "Hong Kong": "香港",
	"Singapore": "新加坡", "Malaysia": "马来西亚", "Indonesia": "印度尼西亚", "India": "印度",
	"Philippines": "菲律宾", "Australia": "澳大利亚", "New Zealand": "新西兰", "Canada": "加拿大",
	"Ukraine": "乌克兰", "France": "法国", "Germany": "德国", "Netherlands": "荷兰", "Sweden": "瑞典",
	"Norway": "挪威", "Spain": "西班牙", "Turkey": "土耳其", "South Africa": "南非", "Brazil": "巴西",
	"Argentina": "阿根廷", "Chile": "智利", "Mexico": "墨西哥", "Romania": "罗马尼亚", "Poland": "波兰",
	"Italy": "意大利", "Switzerland": "瑞士", "Belgium": "比利时", "Austria": "奥地利", "Denmark": "丹麦",
	"Finland": "芬兰", "Portugal": "葡萄牙", "Greece": "希腊", "Ireland": "爱尔兰", "Israel": "以色列",
	"United Arab Emirates": "阿联酋", "Macao": "澳门", "Macau": "澳门",
}

// NormalizeCountry maps common English country names to their Chinese form so
// filters match regardless of which label a node carries.
func NormalizeCountry(value string) string {
	v := strings.TrimSpace(value)
	if t, ok := countryTranslations[v]; ok {
		return t
	}
	return v
}
