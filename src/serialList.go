package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
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
	showAll bool
)

func init() {
	flag.BoolVar(&showAll, "a", false, "Show all ports including not ready")
	flag.Parse()
}

func checkPortReady(portName string) bool {
	// Windowsのデバイスパス形式に変換
	portPath := windows.StringToUTF16Ptr(`\\.\` + portName)

	// ポートを開く
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

		// DCBの取得を試みる（実際の通信は行わない）
		var dcb windows.DCB
		err = windows.GetCommState(handle, &dcb)
		return err == nil
	}

	func main() {
		commKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DEVICEMAP\SERIALCOMM`,
		registry.READ)
		if err != nil {
			log.Fatal(err)
		}
		defer commKey.Close()

		usbKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Enum\USB`,
		registry.READ)
		if err != nil {
			log.Fatal(err)
		}
		defer usbKey.Close()

		devices := make(map[string]*DeviceInfo)
		findAllSerialPorts(commKey, devices)
		findUSBInfo(`SYSTEM\CurrentControlSet\Enum\USB`, usbKey, devices)

		first := true
		for _, device := range devices {
			if showAll || device.Ready {
				if !first {
					fmt.Println()
				}
				first = false

				if device.IsUSB {
					status := "ready"
					if !device.Ready {
						status = "busy"
					}

					vendorInfo := device.VID
					if vendor, exists := knownVendors[device.VID]; exists {
						vendorInfo = fmt.Sprintf("%s %s", device.VID, vendor)
					}

					fmt.Printf("%s [VID:%s PID:%s] : %s", 
					device.PortName, vendorInfo, device.PID, status)
				} else {
					status := "ready"
					if !device.Ready {
						status = "busy"
					}
					fmt.Printf("%s : %s", device.PortName, status)
				}
			}
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
