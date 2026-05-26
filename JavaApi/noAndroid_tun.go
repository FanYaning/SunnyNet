//go:build !android
// +build !android

package JavaJni

const (
	AndroidSDKQ = 29

	IPProtoTCP = 6
	IPProtoUDP = 17
)

var (
	AndroidSDKInt           int
	TunPackageLookupEnabled bool
)

func SdkInt(env Env) int { return 0 }

func InitTunConnectivityManager(env Env, cm Jobject) bool {
	AndroidSDKInt = 0
	TunPackageLookupEnabled = false
	return false
}

func CurrentApplication(env Env) Jobject { return 0 }

func NewInetSocketAddress(env Env, host string, port int) Jobject { return 0 }

func GetConnectionOwnerUid(env Env, cm Jobject, protocol int, local, remote Jobject) int {
	return -1
}

func PackageNameForUid(env Env, uid int) TunPackageResult { return TunPackageResult{} }

type TunPackageResult struct {
	Name       string
	Registered bool
}

func TunGetRequestPackageName(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) TunPackageResult {
	return TunPackageResult{}
}
