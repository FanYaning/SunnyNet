package getPackageName

// Result 为 TUN 五元组包名查询结果（与 JavaApi.TunPackageResult 对应）。
type Result struct {
	Name       string // 来源应用包名（已去掉 getNameForUid 的 ":appId" 后缀）
	Registered bool   // Name 是否为已安装包（pm list packages 可查）；false 时可能为共享 UID 合成名
}
