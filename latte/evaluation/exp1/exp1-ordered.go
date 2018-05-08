package main

import (
	"fmt"

	base "github.com/jerryz920/conferences/latte/evaluation"
)

func Work(client *base.MetadataClient, i int, done chan bool) {
	block := NumVM * levels[NumLevel-1] / Nthread

	start := i * block
	end := (i + 1) * block

	for j := start; j < end; j++ {
		first := j / (128 * 128)
		second := (j / 128) % 128
		third := j % 128
		vmip := fmt.Sprintf("192.%d.%d.%d", first, second, third)
		vmid := fmt.Sprintf("vm%d", j)
		cidr := fmt.Sprintf("%d.%d.%d.0/24", first, second, third)
		vpc := fmt.Sprintf("vpc%d", i)

		client.Request("/postVMInstance", IaaS,
			vmid,
			"image-vm",
			vmip,
			cidr,
			vpc,
			"noauth:vm")
		confs := make([]string, 10)
		for x := 0; x < 10; x++ {
			confs[x] = fmt.Sprintf("ccc%d", x)
		}
		client.Request("/postInstanceConfig", IaaS, confs...)

		/// 50 ctns
		for k := 0; k < 50; k++ {

			ctnip := fmt.Sprintf("%d.%d.%d.%d", first, second, third, k)
			ctnid := fmt.Sprintf("vm%d-ctn%d", j, k)
			client.Request("/postInstance", vmid,
				ctnip,
				"image-ctn",
				ctnid,
				"noauth:docker")
			confs2 := make([]string, 20)
			for x := 0; x < 20; x++ {
				confs2[x] = fmt.Sprintf("cccc%d", x)
			}
			client.Request("/postInstanceConfig", IaaS, confs2...)

			// 4 procs
			for l := 0; l < 4; l++ {
				port1 := 30000 + l*1000
				port2 := 30999 + l*1000
				pip := fmt.Sprintf("%d.%d.%d.%d:%d-%d", first, second, third, k,
					port1, port2)
				pid := fmt.Sprintf("vm%d-ctn%d-spark%d", j, k, l)
				client.Request("/postInstance", ctnid,
					pid,
					"image-spark",
					pip,
					"noauth:noauth:spark")
				confs3 := make([]string, 20)
				for x := 0; x < 20; x++ {
					confs3[x] = fmt.Sprintf("cccc%d", x)
				}
				client.Request("/postInstanceConfig", IaaS, confs3...)
			}
		}
	}
	for j := start; j < end; j++ {
		first := j / (128 * 128)
		second := (j / 128) % 128
		third := j % 128
		vmip := fmt.Sprintf("192.%d.%d.%d", first, second, third)
		client.Request("/checkFetch", "vmcheck", fmt.Sprintf("%s:10000", vmip))
		/// 50 ctns
		for k := 0; k < 50; k++ {

			ctnip := fmt.Sprintf("%d.%d.%d.%d", first, second, third, k)
			client.Request("/checkFetch", "ctncheck", fmt.Sprintf("%s:10000", ctnip))

			// 4 procs
			for l := 0; l < 4; l++ {
				port1 := 30000 + l*1000
				client.Request("/checkFetch", "proccheck", fmt.Sprintf("%s:%d", ctnip, port1+1))
			}
		}
	}

	for j := start; j < end; j++ {
		vmid := fmt.Sprintf("vm%d", j)
		/// 50 ctns
		for k := 0; k < 50; k++ {
			ctnid := fmt.Sprintf("vm%d-ctn%d", j, k)
			// 4 procs
			for l := 0; l < 4; l++ {
				pid := fmt.Sprintf("vm%d-ctn%d-spark%d", j, k, l)
				client.Request("/lazyDeleteInstance", ctnid, pid)
			}
			client.Request("/lazyDeleteInstance", vmid, ctnid)
		}

		client.Request("/lazyDeleteInstance", IaaS, vmid)
	}

	done <- true
}
