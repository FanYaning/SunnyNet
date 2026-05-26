//go:build !android
// +build !android

package getPackageName

const (
	ProtocolTCP = 6
	ProtocolUDP = 17
)

// SetConnectivityManager 非 Android 平台无 TUN 包名查询能力。
func SetConnectivityManager(uintptr) {}

// queryPackageName 非 Android 恒为空结果。
func queryPackageName(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) Result {
	return Result{}
}
