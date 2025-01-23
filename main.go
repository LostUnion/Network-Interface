package main

import (
	// "log"
	"fmt"
	"syscall"
	"unsafe"
	"time"
	"runtime"


	"golang.org/x/sys/windows"
)

var (
	wintun_module                 = syscall.NewLazyDLL("wintun.dll")
	iphlpapiDLL                   = syscall.NewLazyDLL("iphlpapi.dll")
	procGetAdaptersAddresses 	  = iphlpapiDLL.NewProc("GetAdaptersAddresses")
	wintunCreateAdapter           = wintun_module.NewProc("WintunCreateAdapter")
	wintunOpenAdapter			  = wintun_module.NewProc("WintunOpenAdapter")
	wintunCloseAdapter            = wintun_module.NewProc("WintunCloseAdapter")
	wintunGetRunningDriverVersion = wintun_module.NewProc("WintunGetRunningDriverVersion")
	wintunGetAdapterLuid          = wintun_module.NewProc("WintunGetAdapterLUID")
	wintunEnumAdapters            = wintun_module.NewProc("WintunEnumAdapters")
	wintunDeleteDriver			  = wintun_module.NewProc("WintunDeleteDriver")

	createIpAddressEntry = iphlpapiDLL.NewProc("CreateUnicastIpAddressEntry")

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

// Packet with data
type Packet struct {
	Next *Packet              // Pointer to next packet in queue
	Size uint32               // Size of packet (max WINTUN_MAX_IP_PACKET_SIZE)
	Data *[PacketSizeMax]byte // Pointer to layer 3 IPv4 or IPv6 packet
}

// const (
// 	AF_UNSPEC = 0
// 	GAA_FLAG_INCLUDE_PREFIX = 0x0010
// )

// type SOCKET_ADDRESS struct {
// 	LpSockaddr *syscall.RawSockaddrAny
// 	Iaddrlen   int32
// }

// type IP_ADAPTER_UNICAST_ADDRESS struct {
// 	Length             uint32
// 	Flags              uint32
// 	Next               *IP_ADAPTER_UNICAST_ADDRESS
// 	Address            SOCKET_ADDRESS
// 	PrefixOrigin       int32
// 	SuffixOrigin       int32
// 	DadState           int32
// 	ValidLifetime      uint32
// 	PreferredLifetime  uint32
// 	LeaseLifetime      uint32
// 	OnLinkPrefixLength uint8
// }

// type IP_ADAPTER_ANYCAST_ADDRESS struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_ANYCAST_ADDRESS
// 	Address SOCKET_ADDRESS
// }

// type IP_ADAPTER_MULTICAST_ADDRESS struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_MULTICAST_ADDRESS
// 	Address SOCKET_ADDRESS
// }

// type IP_ADAPTER_DNS_SERVER_ADDRESS struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_DNS_SERVER_ADDRESS
// 	Address SOCKET_ADDRESS
// }

// type IP_ADAPTER_PREFIX struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_PREFIX
// 	Address SOCKET_ADDRESS
// 	PrefixLength uint32
// }

// type IP_ADAPTER_WINS_SERVER_ADDRESS struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_WINS_SERVER_ADDRESS
// 	Address SOCKET_ADDRESS
// }

// type IP_ADAPTER_GATEWAY_ADDRESS struct {
// 	Length  uint32
// 	Flags   uint32
// 	Next    *IP_ADAPTER_GATEWAY_ADDRESS
// 	Address SOCKET_ADDRESS
// }

// type IP_ADAPTER_ADDRESSES struct {
// 	Length                uint32
// 	IfIndex               uint32
// 	Next                  *IP_ADAPTER_ADDRESSES
// 	AdapterName           *byte
// 	FirstUnicastAddress   *IP_ADAPTER_UNICAST_ADDRESS
// 	FirstAnycastAddress   *IP_ADAPTER_ANYCAST_ADDRESS
// 	FirstMulticastAddress *IP_ADAPTER_MULTICAST_ADDRESS
// 	FirstDnsServerAddress *IP_ADAPTER_DNS_SERVER_ADDRESS
// 	DnsSuffix             *uint16
// 	Description           *uint16
// 	FriendlyName          *uint16
// 	PhysicalAddress       [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
// 	PhysicalAddressLength uint32
// 	Flags                 uint32
// 	Mtu                   uint32
// 	IfType                uint32
// 	OperStatus            uint32
// 	Ipv6IfIndex           uint32
// 	ZoneIndices           [16]uint32
// 	FirstPrefix           *IP_ADAPTER_PREFIX
// 	TransmitLinkSpeed     uint64
// 	ReceiveLinkSpeed      uint64
// 	FirstWinsServerAddress *IP_ADAPTER_WINS_SERVER_ADDRESS
// 	FirstGatewayAddress   *IP_ADAPTER_GATEWAY_ADDRESS
// 	Ipv4Metric            uint32
// 	Ipv6Metric            uint32
// 	Luid                  uint64
// 	Dhcpv4Server          SOCKET_ADDRESS
// 	CompartmentId         uint32
// 	NetworkGuid           windows.GUID
// 	ConnectionType        uint32
// 	TunnelType            uint32
// 	Dhcpv6Server          SOCKET_ADDRESS
// 	Dhcpv6ClientDuid      []byte
// 	Dhcpv6ClientDuidLength uint32
// 	Dhcpv6Iaid            uint32
// }

func closeAdapter(main *Adapter) {
	syscall.Syscall(wintunCloseAdapter.Addr(), 1, main.handle, 0, 0)
}

func CreateAdapter(name string, tunnelType string, requestedGUID *windows.GUID) (wintun *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	var tunnelType16 *uint16
	tunnelType16, err = windows.UTF16PtrFromString(tunnelType)
	if err != nil {
		return
	}
	r0, _, e1 := syscall.Syscall(wintunCreateAdapter.Addr(), 3, uintptr(unsafe.Pointer(name16)), uintptr(unsafe.Pointer(tunnelType16)), uintptr(unsafe.Pointer(requestedGUID)))
	if r0 == 0 {
		err = e1
		return
	}

	
	wintun = &Adapter{handle: r0}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

// OpenAdapter opens an existing Wintun adapter by name.
func OpenAdapter(name string) (wintun *Adapter, err error) {
	var name16 *uint16
	name16, err = windows.UTF16PtrFromString(name)
	if err != nil {
		return
	}
	r0, _, e1 := syscall.Syscall(wintunOpenAdapter.Addr(), 1, uintptr(unsafe.Pointer(name16)), 0, 0)
	if r0 == 0 {
		err = e1
		return
	}
	wintun = &Adapter{handle: r0}
	runtime.SetFinalizer(wintun, closeAdapter)
	return
}

func (wintun *Adapter) Close() (err error) {
	runtime.SetFinalizer(wintun, nil)
	r1, _, e1 := syscall.Syscall(wintunCloseAdapter.Addr(), 1, wintun.handle, 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

func Uninstall() (err error) {
	r1, _, e1 := syscall.Syscall(wintunDeleteDriver.Addr(), 0, 0, 0, 0)
	if r1 == 0 {
		err = e1
	}
	return
}

func RunningVersion() (version uint32, err error) {
	r0, _, e1 := syscall.Syscall(wintunGetRunningDriverVersion.Addr(), 0, 0, 0, 0)
	version = uint32(r0)
	if version == 0 {
		err = e1
	}
	return
}

func (wintun *Adapter) LUID() (luid uint64) {
	syscall.Syscall(wintunGetAdapterLuid.Addr(), 2, uintptr(wintun.handle), uintptr(unsafe.Pointer(&luid)), 0)
	return
}

func (wintun *Adapter) StartSession(capacity uint32) (session Session, err error) {
	r0, _, e1 := syscall.Syscall(wintunStartSession.Addr(), 2, uintptr(wintun.handle), uintptr(capacity), 0)
	if r0 == 0 {
		err = e1
	} else {
		session = Session{r0}
	}
	return
}

func (session Session) End() {
	syscall.Syscall(wintunEndSession.Addr(), 1, session.handle, 0, 0)
	session.handle = 0
}

func main() {
	var guid windows.GUID
	adapter, err := CreateAdapter("TestNetwork", "Wintun", &guid)
	fmt.Println(adapter, err)

	time.Sleep(1 * time.Second)

	version, err := RunningVersion()
	fmt.Println(version, err)

	time.Sleep(1 * time.Second)

	open_adapter, err := OpenAdapter("TestNetwork")
	fmt.Println(open_adapter, err)

	time.Sleep(1 * time.Second)

	start_session_open_adapter, err := open_adapter.StartSession(0x4000000)
	fmt.Println(start_session_open_adapter, err)

	time.Sleep(1 * time.Second)

	res := adapter.Close()
	o_res := open_adapter.Close()
	fmt.Println(res, o_res)
	Uninstall()
	
}