package main

import (
	"flag"
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"log"
	"strings"
)

const (
	registryUSBPath    = `SYSTEM\CurrentControlSet\Enum\USB`
	registrySerialPath = `HARDWARE\DEVICEMAP\SERIALCOMM`
)

type DeviceInfo struct {
	PortName string
	VID      string
	PID      string
	Ready    bool
	IsUSB    bool
}

var knownVendors = map[string]string{
	"2E8A": "Raspberry Pi Foundation",
}

var (
	noPause bool
)

func init() {
	flag.BoolVar(&noPause, "n", false, "Exit without waiting for key press.")

	flag.Usage = func() {
		fmt.Println("Display the status of COM ports.")
		fmt.Println("Usage of program:")
		fmt.Println("-n    Exit without waiting for key press.")
		flag.PrintDefaults()
	}

	flag.Parse()
}

func main() {
	devices, err := findDevices()
	if err != nil {
		log.Fatal(err)
	}

	displayDevices(devices)
	if !noPause {
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
	}
}

func findDevices() (map[string]*DeviceInfo, error) {
	// レジストリキーを開く
	keys, err := openRegistryKeys()
	if err != nil {
		return nil, err
	}
	defer closeRegistryKeys(keys)

	// デバイス一覧を取得
	devices := make(map[string]*DeviceInfo)
	findAllSerialPorts(keys.commKey, devices)
	findUSBInfo(registryUSBPath, keys.usbKey, devices)
	return devices, nil
}

type registryKeys struct {
	commKey registry.Key
	usbKey  registry.Key
}

func openRegistryKeys() (*registryKeys, error) {
	commKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		registrySerialPath,
		registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open SERIALCOMM registry: %v", err)
	}

	usbKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		registryUSBPath,
		registry.READ)
	if err != nil {
		commKey.Close()
		return nil, fmt.Errorf("failed to open USB registry: %v", err)
	}

	return &registryKeys{
		commKey: commKey,
		usbKey:  usbKey,
	}, nil
}

func closeRegistryKeys(keys *registryKeys) {
	if keys != nil {
		keys.commKey.Close()
		keys.usbKey.Close()
	}
}

func findAllSerialPorts(key registry.Key, devices map[string]*DeviceInfo) {
	values, err := key.ReadValueNames(-1)
	if err != nil {
		return
	}

	for _, name := range values {
		portName, _, err := key.GetStringValue(name)
		if err != nil {
			continue
		}
		ready := checkPortReady(portName)
		devices[portName] = &DeviceInfo{
			PortName: portName,
			Ready:    ready,
			IsUSB:    false,
		}
	}
}

func findUSBInfo(basePath string, key registry.Key, devices map[string]*DeviceInfo) {
	subKeys, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return
	}

	for _, subKey := range subKeys {
		if strings.Contains(subKey, "VID_") {
			path := basePath + "\\" + subKey
			deviceKey, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.READ)
			if err != nil {
				continue
			}
			defer deviceKey.Close()

			subDevices, err := deviceKey.ReadSubKeyNames(-1)
			if err != nil {
				continue
			}
			for _, subDevice := range subDevices {
				subPath := path + "\\" + subDevice
				subDeviceKey, err := registry.OpenKey(registry.LOCAL_MACHINE, subPath, registry.READ)
				if err != nil {
					continue
				}
				defer subDeviceKey.Close()

				friendlyName, _, err := subDeviceKey.GetStringValue("FriendlyName")
				if err != nil {
					continue
				}

				if strings.Contains(friendlyName, "COM") {
					portName := extractCOMPort(friendlyName)
					if device, exists := devices[portName]; exists {
						vid, pid := extractVIDPID(subKey)
						device.VID = vid
						device.PID = pid
						device.IsUSB = true
					}
				}
			}
		}
	}
}

func checkPortReady(portName string) bool {
	portPath := windows.StringToUTF16Ptr(`\\.\` + portName)

	handle, err := windows.CreateFile(
		portPath,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0)

	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var dcb windows.DCB
	err = windows.GetCommState(handle, &dcb)
	return err == nil
}

func displayDevices(devices map[string]*DeviceInfo) {
	first := true
	for _, device := range devices {
		if !first {
			fmt.Println()
		}
		first = false
		displayDevice(device)
	}
}

func displayDevice(device *DeviceInfo) {
	if device.IsUSB {
		status := getDeviceStatus(device.Ready)
		vendorInfo := getVendorInfo(device.VID)
		fmt.Printf("%s [VID:%s PID:%s] : %s",
			device.PortName, vendorInfo, device.PID, status)
	} else {
		status := getDeviceStatus(device.Ready)
		fmt.Printf("%s : %s", device.PortName, status)
	}
}

func getDeviceStatus(ready bool) string {
	if ready {
		return "ready"
	}
	return "busy"
}

func getVendorInfo(vid string) string {
	if vendor, exists := knownVendors[vid]; exists {
		return fmt.Sprintf("%s %s", vid, vendor)
	}
	return vid
}

func extractCOMPort(friendlyName string) string {
	start := strings.Index(friendlyName, "(COM")
	if start == -1 {
		return ""
	}
	end := strings.Index(friendlyName[start:], ")")
	if end == -1 {
		return ""
	}
	return friendlyName[start+1 : start+end]
}

func extractVIDPID(devicePath string) (string, string) {
	devicePath = strings.ToUpper(devicePath)
	vid := ""
	pid := ""

	vidIndex := strings.Index(devicePath, "VID_")
	if vidIndex != -1 && len(devicePath) >= vidIndex+8 {
		vid = devicePath[vidIndex+4 : vidIndex+8]
	}

	pidIndex := strings.Index(devicePath, "PID_")
	if pidIndex != -1 && len(devicePath) >= pidIndex+8 {
		pid = devicePath[pidIndex+4 : pidIndex+8]
	}

	return vid, pid
}
