package main

import (
	"os"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	wintun_module                 = syscall.NewLazyDLL("wintun.dll")
	wintunCreateAdapter           = wintun_module.NewProc("WintunCreateAdapter")
	wintunOpenAdapter             = wintun_module.NewProc("WintunOpenAdapter")
	wintunCloseAdapter            = wintun_module.NewProc("WintunCloseAdapter")
	wintunGetRunningDriverVersion = wintun_module.NewProc("WintunGetRunningDriverVersion")
	wintunGetAdapterLuid          = wintun_module.NewProc("WintunGetAdapterLUID")
	wintunEnumAdapters            = wintun_module.NewProc("WintunEnumAdapters")
	wintunDeleteDriver            = wintun_module.NewProc("WintunDeleteDriver")

	wintunAllocateSendPacket   = wintun_module.NewProc("WintunAllocateSendPacket")
	wintunEndSession           = wintun_module.NewProc("WintunEndSession")
	wintunGetReadWaitEvent     = wintun_module.NewProc("WintunGetReadWaitEvent")
	wintunReceivePacket        = wintun_module.NewProc("WintunReceivePacket")
	wintunReleaseReceivePacket = wintun_module.NewProc("WintunReleaseReceivePacket")
	wintunSendPacket           = wintun_module.NewProc("WintunSendPacket")
	wintunStartSession         = wintun_module.NewProc("WintunStartSession")
)

const AdapterNameMax = 128

type Adapter struct {
	handle uintptr
}

type Session struct {
	handle uintptr
}

const (
	PacketSizeMax   = 0xffff    // Maximum packet size
	RingCapacityMin = 0x20000   // Minimum ring capacity (128 kiB)
	RingCapacityMax = 0x4000000 // Maximum ring capacity (64 MiB)
)

type Packet struct {
	Next *Packet              // Pointer to next packet in queue
	Size uint32               // Size of packet (max WINTUN_MAX_IP_PACKET_SIZE)
	Data *[PacketSizeMax]byte // Pointer to layer 3 IPv4 or IPv6 packet
}

func closeAdapter(wintun *Adapter) {
	syscall.Syscall(wintunCloseAdapter.Addr(), 1, wintun.handle, 0, 0)
}

func CreateAdapter(
	adapter_name string,
	tunnel_Type string,
	GUID *windows.GUID) (wintun *Adapter,
		   				 err error) {

	var adapter_name_16 *uint16

	adapter_name_16, err = windows.UTF16PtrFromString(adapter_name)
	if err != nil {
		return
	}

	var tunnel_Type_16 *uint16
	tunnel_Type_16, err = windows.UTF16PtrFromString(tunnel_Type)
	if err != nil {
		return
	}

	result, _, err_ := syscall.Syscall(
		wintunCreateAdapter.Addr(),
		3,
		uintptr(unsafe.Pointer(adapter_name_16)),
		uintptr(unsafe.Pointer(tunnel_Type_16)),
		uintptr(unsafe.Pointer(GUID)),
	)

	if result == 0 {
		err = err_
		return
	}

	wintun = &Adapter{handle: result}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

// OpenAdapter opens an existing Wintun adapter by name.
func OpenAdapter(adapter_name string) (wintun *Adapter, err error) {

	var adapter_name_16 *uint16

	adapter_name_16, err = windows.UTF16PtrFromString(adapter_name)
	if err != nil {
		return
	}

	result, _, err_ := syscall.Syscall(
		wintunOpenAdapter.Addr(),
		1,
		uintptr(unsafe.Pointer(adapter_name_16)),
		0,
		0,
	)

	if result == 0 {
		err = err_
		return
	}

	wintun = &Adapter{handle: result}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

func (wintun *Adapter) Close() (err error) {
	runtime.SetFinalizer(wintun, nil)

	result, _, err_ := syscall.Syscall(
		wintunCloseAdapter.Addr(),
		1,
		wintun.handle,
		0,
		0,
	)

	if result == 0 {
		err = err_
	}
	return
}

func Uninstall() (err error) {
	result, _, err_ := syscall.Syscall(
		wintunDeleteDriver.Addr(),
		0,
		0,
		0,
		0,
	)

	if result == 0 {
		err = err_
	}
	return
}

func RunningVersion() (version uint32, err error) {
	result, _, err_ := syscall.Syscall(
		wintunGetRunningDriverVersion.Addr(),
		0,
		0,
		0,
		0,
	)

	version = uint32(result)

	if version == 0 {
		err = err_
	}
	return
}

func (wintun *Adapter) LUID() (luid uint64) {
	syscall.Syscall(
		wintunGetAdapterLuid.Addr(),
		2,
		uintptr(wintun.handle),
		uintptr(unsafe.Pointer(&luid)),
		0,
	)
	return
}

func (wintun *Adapter) StartSession(capacity uint32) (session Session, err error) {
	result, _, err_ := syscall.Syscall(
		wintunStartSession.Addr(),
		2,
		uintptr(wintun.handle),
		uintptr(capacity),
		0,
	)

	if result == 0 {
		err = err_
	} else {
		session = Session{result}
	}
	return
}

func (session Session) End() {
	syscall.Syscall(
		wintunEndSession.Addr(),
		1,
		session.handle,
		0,
		0,
	)

	session.handle = 0
}

func main() {


	var guid windows.GUID

	adapter_name := "Test VPN Service"
	adapter_type := "Wintun"

	fmt.Printf("[TUN] Creating a network interface named \"%s\"...\n", adapter_name)
	adapter, err := CreateAdapter(adapter_name, adapter_type, &guid)
	if err != nil {
		fmt.Printf("[TUN] An error occurred when creating the \"%s\" interface.\n[Error] %s\n", adapter_name, err)
		fmt.Printf("[FATAL ERROR][TUN] func \"CreateAdapter\" failed. Exiting the program.\n")
		if adapter != nil {
			adapter.Close()
		}
		os.Exit(1)
	} else {
		fmt.Printf("[TUN][INTF: %d] The \"%s\" interface was created successfully.\n", adapter.handle, adapter_name)
	}

	time.Sleep(1 * time.Second)

	version_driver, err := RunningVersion()
	if err != nil {
		fmt.Printf("[TUN] An error occurred while getting the Wintun driver version.\n[Error] %s\n", err)
		fmt.Printf("[ERROR][TUN] func \"RunningVersion\" failed.\n")
	} else {
		fmt.Printf("[TUN][INTF: %d] Installed Wintun driver version %d.\n", adapter.handle, version_driver)
	}

	// time.Sleep(1 * time.Second)

	// open_adapter, err := OpenAdapter(adapter_name)
	// fmt.Println(open_adapter, err)

	time.Sleep(1 * time.Second)

	fmt.Printf("[TUN][INTF: %d] Initializing session startup on interface.\n", adapter.handle)
	start_session, err := adapter.StartSession(RingCapacityMax)
	if err != nil {
		fmt.Printf("[TUN][INTF: %d] It is not possible to start a session on the interface.\n[Error] %s\n", adapter.handle, err)
		fmt.Printf("[FATAL ERROR][TUN] func \"StartSession\" failed. Exiting the program.\n")
		if adapter != nil {
			adapter.Close()
		}
		if start_session.handle != 0 {
			start_session.End()
		}
		os.Exit(1)
	} else {
		fmt.Printf("[TUN][INTF: %d][SESS: %d] Session is running on the interface.\n", adapter.handle, start_session.handle)
	}

	time.Sleep(1 * time.Second)

	fmt.Printf("[TUN][INTF: %d][SESS: %d] Initializing the session stop in the interface.\n", adapter.handle, start_session.handle)
	start_session.End()
	fmt.Printf("[TUN][INTF: %d] The session was successfully closed.\n", adapter.handle)

	time.Sleep(1 * time.Second)

	fmt.Printf("[TUN][INTF: %d] Initializing the stop the interface.\n", adapter.handle)
	close_err := adapter.Close()
	if close_err != nil {
		fmt.Printf("[TUN][INTF: %d] Interface shutdown failed.\n[Error] %s\n", adapter.handle, close_err)
		fmt.Printf("[FATAL ERROR][TUN] func \"Close\" failed. Exiting the program.\n")
	} else {
		fmt.Printf("[TUN] The interface is successfully closed.\n")
	}

	time.Sleep(1 * time.Second)

	fmt.Printf("[TUN] Initialized removal of the wintun driver.\n")
	unistall_err := Uninstall()
	if unistall_err != nil {
		fmt.Printf("[TUN] The driver has not been deleted.\n[Error] %s\n", unistall_err)
		fmt.Printf("[FATAL ERROR][TUN] func \"Uninstall\" failed. Exiting the program.\n")
	} else {
		fmt.Printf("[TUN] The driver was successfully deleted.")
	}

}
