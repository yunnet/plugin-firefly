package firefly

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"testing"
)

func Test_Cpu(t *testing.T) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("cpu:", cpuInfo)
}

func Test_disk(t *testing.T) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("partitions:", partitions)

	diskUsages := make([]*disk.UsageStat, len(partitions))
	for k, v := range partitions {
		usageStat, err := disk.Usage(v.Mountpoint)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		diskUsages[k] = usageStat
	}
	fmt.Println("disk:", diskUsages)

	// C
	usageStat, _ := disk.Usage("c:")
	fmt.Println("disk: C:", usageStat)

	// D
	usageStat2, _ := disk.Usage("d:")
	fmt.Println("disk: D:", usageStat2)
}
