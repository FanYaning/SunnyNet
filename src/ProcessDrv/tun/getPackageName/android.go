//go:build android
// +build android

package getPackageName

import JavaJni "github.com/qtgolang/SunnyNet/JavaApi"

const (
	ProtocolTCP = 6  // IPPROTO_TCP
	ProtocolUDP = 17 // IPPROTO_UDP
)

// SetConnectivityManager 兼容旧接口；实际 CM 由 JavaJni.InitTunConnectivityManager 在 OnTunSetFd 中注入。
// 调用时清空包名缓存，避免 VPN 重建后沿用旧五元组结果。
func SetConnectivityManager(uintptr) {
	ClearPackageNameCache()
}

// queryPackageName 执行一次 JNI 查询（无缓存）；由 GetRequestPackageName 统一加缓存。
func queryPackageName(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) Result {
	r := JavaJni.TunGetRequestPackageName(protocol, srcIp, srcPort, dstIp, dstPort)
	return Result{Name: r.Name, Registered: r.Registered}
}
