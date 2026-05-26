//go:build android
// +build android

package JavaJni

import (
	"runtime"
	"strings"
	"sync"
)

const (
	AndroidSDKQ = 29

	IPProtoTCP = 6
	IPProtoUDP = 17

	invalidUID = -1
)

// TunPackageResult TUN 五元组查询到的包名及是否在 PackageManager 注册。
type TunPackageResult struct {
	Name       string
	Registered bool
}

var (
	// AndroidSDKInt 在 InitTunConnectivityManager 时写入当前系统 SDK。
	AndroidSDKInt int
	// TunPackageLookupEnabled 为 true 表示已保存 ConnectivityManager 且 SDK>=29。
	TunPackageLookupEnabled bool

	tunCM   Jobject
	tunCMMu sync.RWMutex
)

// SdkInt 读取 android.os.Build.VERSION.SDK_INT。
func SdkInt(env Env) int {
	verClass := env.FindClass("android/os/Build$VERSION")
	if verClass == 0 {
		return 0
	}
	defer env.DeleteLocalRef(verClass)
	field := env.GetStaticFieldID(verClass, "SDK_INT", "I")
	if field == 0 {
		return 0
	}
	return env.GetStaticIntField(verClass, field)
}

// InitTunConnectivityManager 在 OnTunSetFd 中调用：仅 Android 10+ 保存 CM 全局引用并启用包名查询。
func InitTunConnectivityManager(env Env, cm Jobject) bool {
	sdk := SdkInt(env)
	AndroidSDKInt = sdk
	if sdk < AndroidSDKQ || cm == 0 {
		clearTunConnectivityManager(env)
		return false
	}
	clearTunConnectivityManager(env)
	tunCMMu.Lock()
	tunCM = env.NewGlobalRef(cm)
	tunCMMu.Unlock()
	TunPackageLookupEnabled = true
	return true
}

func clearTunConnectivityManager(env Env) {
	tunCMMu.Lock()
	old := tunCM
	tunCM = 0
	tunCMMu.Unlock()
	if old != 0 {
		env.DeleteGlobalRef(old)
	}
	TunPackageLookupEnabled = false
}

// CurrentApplication 返回 ActivityThread.currentApplication()。
func CurrentApplication(env Env) Jobject {
	atClass := env.FindClass("android/app/ActivityThread")
	if atClass == 0 {
		return 0
	}
	defer env.DeleteLocalRef(atClass)

	method := env.GetStaticMethodID(atClass, "currentApplication", "()Landroid/app/Application;")
	if method == 0 {
		return 0
	}
	app := env.CallStaticObjectMethodA(atClass, method)
	if app == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return 0
	}
	return app
}

// NewInetSocketAddress 构造 java.net.InetSocketAddress(InetAddress, port)。
func NewInetSocketAddress(env Env, host string, port int) Jobject {
	if host == "" || port <= 0 || port > 65535 {
		return 0
	}
	inetClass := env.FindClass("java/net/InetAddress")
	if inetClass == 0 {
		return 0
	}
	defer env.DeleteLocalRef(inetClass)

	getByName := env.GetStaticMethodID(inetClass, "getByName", "(Ljava/lang/String;)Ljava/net/InetAddress;")
	if getByName == 0 {
		return 0
	}
	hostJ := env.NewString(host)
	if hostJ == 0 {
		return 0
	}
	defer env.DeleteLocalRef(hostJ)

	inetAddr := env.CallStaticObjectMethodA(inetClass, getByName, Jvalue(hostJ))
	if inetAddr == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return 0
	}
	defer env.DeleteLocalRef(inetAddr)

	isaClass := env.FindClass("java/net/InetSocketAddress")
	if isaClass == 0 {
		return 0
	}
	defer env.DeleteLocalRef(isaClass)

	init := env.GetMethodID(isaClass, "<init>", "(Ljava/net/InetAddress;I)V")
	if init == 0 {
		return 0
	}
	sock := env.NewObjectA(isaClass, init, Jvalue(inetAddr), Jvalue(port))
	if sock == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return 0
	}
	return sock
}

// GetConnectionOwnerUid 调用 ConnectivityManager.getConnectionOwnerUid。
func GetConnectionOwnerUid(env Env, cm Jobject, protocol int, local, remote Jobject) int {
	cmClass := env.FindClass("android/net/ConnectivityManager")
	if cmClass == 0 {
		return invalidUID
	}
	defer env.DeleteLocalRef(cmClass)

	method := env.GetMethodID(cmClass, "getConnectionOwnerUid",
		"(ILjava/net/InetSocketAddress;Ljava/net/InetSocketAddress;)I")
	if method == 0 {
		return invalidUID
	}

	uid := env.CallIntMethodA(cm, method, Jvalue(protocol), Jvalue(local), Jvalue(remote))
	if env.ExceptionCheck() {
		env.ExceptionClear()
		return invalidUID
	}
	return uid
}

// normalizePackageName 去掉 getNameForUid 返回的 ":appId" 后缀，得到纯包名。
func normalizePackageName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if i := strings.LastIndex(name, ":"); i > 0 {
		suffix := name[i+1:]
		if suffix != "" && isDecimalString(suffix) {
			return name[:i]
		}
	}
	return name
}

func isDecimalString(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isPackageRegistered 判断包名是否已安装（PackageManager.getPackageInfo 能查到）。
func isPackageRegistered(env Env, pm Jobject, pmClass Jclass, pkg string) bool {
	if pkg == "" {
		return false
	}
	getPI := env.GetMethodID(pmClass, "getPackageInfo", "(Ljava/lang/String;I)Landroid/content/pm/PackageInfo;")
	if getPI == 0 {
		return false
	}
	pkgJ := env.NewString(pkg)
	if pkgJ == 0 {
		return false
	}
	defer env.DeleteLocalRef(pkgJ)
	pi := env.CallObjectMethodA(pm, getPI, Jvalue(pkgJ), Jvalue(0))
	if pi == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return false
	}
	env.DeleteLocalRef(pi)
	return true
}

// packageNameFromUid 优先 getPackagesForUid，否则 getNameForUid 并规范化；第二返回值表示是否在 PM 注册。
func packageNameFromUid(env Env, pm Jobject, pmClass Jclass, uid int) (string, bool) {
	getPkgs := env.GetMethodID(pmClass, "getPackagesForUid", "(I)[Ljava/lang/String;")
	if getPkgs != 0 {
		arr := env.CallObjectMethodA(pm, getPkgs, Jvalue(uid))
		if arr != 0 && !env.ExceptionCheck() {
			defer env.DeleteLocalRef(arr)
			n := env.GetArrayLength(Jarray(arr))
			for i := 0; i < n; i++ {
				item := env.GetObjectArrayElement(JobjectArray(arr), i)
				if item == 0 {
					continue
				}
				pkg := strings.TrimSpace(env.GetString(Jstring(item)))
				env.DeleteLocalRef(item)
				if pkg != "" {
					return pkg, true
				}
			}
		} else {
			env.ExceptionClear()
		}
	}

	getName := env.GetMethodID(pmClass, "getNameForUid", "(I)Ljava/lang/String;")
	if getName == 0 {
		return "", false
	}
	nameObj := env.CallObjectMethodA(pm, getName, Jvalue(uid))
	if nameObj == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return "", false
	}
	defer env.DeleteLocalRef(nameObj)
	pkg := normalizePackageName(env.GetString(Jstring(nameObj)))
	if pkg == "" {
		return "", false
	}
	return pkg, isPackageRegistered(env, pm, pmClass, pkg)
}

// PackageNameForUid 解析 UID 对应纯包名及是否已注册。
func PackageNameForUid(env Env, uid int) TunPackageResult {
	ctx := CurrentApplication(env)
	if ctx == 0 {
		return TunPackageResult{}
	}
	defer env.DeleteLocalRef(ctx)

	getPM := env.GetMethodID(env.GetObjectClass(ctx), "getPackageManager", "()Landroid/content/pm/PackageManager;")
	if getPM == 0 {
		return TunPackageResult{}
	}
	pm := env.CallObjectMethodA(ctx, getPM)
	if pm == 0 || env.ExceptionCheck() {
		env.ExceptionClear()
		return TunPackageResult{}
	}
	defer env.DeleteLocalRef(pm)

	pmClass := env.GetObjectClass(pm)
	defer env.DeleteLocalRef(pmClass)

	name, reg := packageNameFromUid(env, pm, pmClass, uid)
	return TunPackageResult{Name: name, Registered: reg}
}

// TunGetRequestPackageName 根据五元组查询来源包名及是否在 PackageManager 注册。
func TunGetRequestPackageName(protocol int, srcIp string, srcPort int, dstIp string, dstPort int) TunPackageResult {
	if !TunPackageLookupEnabled || GlobalVM == 0 {
		return TunPackageResult{}
	}
	if protocol != IPProtoTCP && protocol != IPProtoUDP {
		return TunPackageResult{}
	}

	tunCMMu.RLock()
	cm := tunCM
	tunCMMu.RUnlock()
	if cm == 0 {
		return TunPackageResult{}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	env, ret := GlobalVM.AttachCurrentThread()
	if ret != JNI_OK {
		return TunPackageResult{}
	}
	defer GlobalVM.DetachCurrentThread()

	local := NewInetSocketAddress(env, srcIp, srcPort)
	if local == 0 {
		return TunPackageResult{}
	}
	defer env.DeleteLocalRef(local)

	remote := NewInetSocketAddress(env, dstIp, dstPort)
	if remote == 0 {
		return TunPackageResult{}
	}
	defer env.DeleteLocalRef(remote)

	uid := GetConnectionOwnerUid(env, cm, protocol, local, remote)
	if uid < 0 {
		return TunPackageResult{}
	}
	return PackageNameForUid(env, uid)
}
