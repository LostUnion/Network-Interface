package main

import (
	"fmt"
	// "os/exec"
	// "strings"
	"syscall"
	"unsafe"
	"time"
)

type WintunAdapterHandle uintptr
type WintunSessionHandle uintptr

var wintunDLL = syscall.NewLazyDLL("wintun.dll")

var (
	wintunCreateAdapter          = wintunDLL.NewProc("WintunCreateAdapter")
	wintunCloseAdapter           = wintunDLL.NewProc("WintunCloseAdapter")
	wintunStartSession           = wintunDLL.NewProc("WintunStartSession")
	wintunEndSession             = wintunDLL.NewProc("WintunEndSession")
	wintunAllocateSendPacket     = wintunDLL.NewProc("WintunAllocateSendPacket")
	wintunSendPacket             = wintunDLL.NewProc("WintunSendPacket")
	wintunReceivePacket          = wintunDLL.NewProc("WintunReceivePacket")
	wintunReleaseReceivePacket   = wintunDLL.NewProc("WintunReleaseReceivePacket")
	wintunGetReadWaitEvent       = wintunDLL.NewProc("WintunGetReadWaitEvent")
	wintunGetRunningDriverVersion = wintunDLL.NewProc("WintunGetRunningDriverVersion")
)

func checkWintunDLL() {
	if wintunDLL == nil {
		fmt.Println("[!] Failed to load wintun.dll")
	} else {
		fmt.Println("[+] wintun.dll loaded successfully")
	}
}

// Создание нового сетевого интерфейса
func createAdapter(adapterName, adapterType string) (WintunAdapterHandle, error) {
	result, _, err := wintunCreateAdapter.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(adapterName,))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(adapterType,))),
	)

	if result == 0 {
		return 0, fmt.Errorf("[!] Failed to create new adapter: %v\n", err)
	}

	return WintunAdapterHandle(result), nil
}

// Запуск сессии
func createSession(adapterHandle WintunAdapterHandle, ringBufferSize uint32) (WintunSessionHandle, error) {
	result, _, err := wintunStartSession.Call(
		uintptr(adapterHandle),
		uintptr(ringBufferSize),
	)

	if result == 0 {
		return 0, fmt.Errorf("[!] Failed to start session on [%d] adapter: %v\n", adapterHandle, err)
	}

	return WintunSessionHandle(result), nil
}

// Остановка сессии
func closeSession(session WintunSessionHandle) (WintunSessionHandle, error) {
	result, _, err := wintunEndSession.Call(
		uintptr(session),
	)

	if result == 0 {
		return 0, fmt.Errorf("[!] Failed to stop session on [%d] adapter: %v\n", session, err)
	}

	return WintunSessionHandle(result), nil
}

// Удаление адаптера
func closeAdapter(adapter WintunAdapterHandle) (WintunAdapterHandle, error) {
	result, _, err := wintunCloseAdapter.Call(
		uintptr(adapter),
	)

	if result == 0 {
		return 0, fmt.Errorf("[!] Failed to create new adapter: %v\n", err)
	}

	return WintunAdapterHandle(result), nil
}

func main() {
	checkWintunDLL()

	adapterName := "Test Network"
	adapterType := "Wintun"
	
	adapter, err := createAdapter(adapterName, adapterType)

	if err != nil {
		fmt.Printf("[!] Error creating adapter %s:%s\n", adapterName, err)
	} else {
		fmt.Printf("[+][%d] Adapter \"%s\" was created successfully.\n", adapter, adapterName)
	}

	result, _, _ := wintunGetRunningDriverVersion.Call()

	if result == 0 {
	    fmt.Println("[!] Error: Wintun driver is not loaded.")
	} else {
	    fmt.Printf("[*][%d] Using wintun Driver Version: %d\n", adapter, result)
	}

	time.Sleep(2 * time.Second)

	session, err := createSession(adapter, 0x400000)

	if err != nil {
		fmt.Printf("[!] Error creating session %s:%s\n", adapterName, err)
	} else {
		fmt.Printf("[+][%d] Session \"%s\" was created successfully.\n", session, adapterName)
	}

	

	time.Sleep(2 * time.Second)
	session_2, err := closeSession(session) // Используйте = вместо := если переменная session уже была определена ранее
	
	if err != nil {
		fmt.Printf("[!] Error closing session %s:%s\n", adapterName, err)
	} else {
		fmt.Printf("[+][%d] Session \"%s\" was closed successfully.\n", session, adapterName)
	}

	time.Sleep(2 * time.Second)
	adapter_2, err := closeAdapter(adapter)

	if err != nil {
		fmt.Printf("[!] Error closing adapter %s:%s\n", adapterName, err)
	} else {
		fmt.Printf("[+][%d] Adapter \"%s\" was closed successfully.\n", adapter, adapterName)
	}

	time.Sleep(2 * time.Second)
	fmt.Println(adapter_2, session_2)

	fmt.Scan(&adapterName)
}