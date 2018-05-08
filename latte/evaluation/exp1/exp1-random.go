package main

import (
	"fmt"

	base "github.com/jerryz920/conferences/latte/evaluation"
)

func WorkRandom(client *base.MetadataClient, i int, done chan bool) {
	block := NumVM * levels[NumLevel-1] / Nthread

	start := i * block
	end := (i + 1) * block

	for j := start; j < end; j++ {
		first := j/(128*128) + 1
		second := (j / 128) % 128
		third := j%128 + 1
		vmip := fmt.Sprintf("192.%d.%d.%d", first, second, third)
		vmid := fmt.Sprintf("vm%d", j)
		cidr := fmt.Sprintf("%d.%d.%d.0/24", first, second, third)
		vpc := fmt.Sprintf("vpc%d", i)

		client.Request(i, "/postVMInstance", IaaS,
			vmid,
			"image-vm",
			fmt.Sprintf("%s:1-65535", vmip),
			cidr,
			vpc,
			"noauth:vm")
		confs := make([]string, 11)
		confs[0] = vmid
		for x := 1; x < 11; x++ {
			confs[x] = fmt.Sprintf("ccc%d", x)
		}
		client.Request(i, "/postInstanceConfig", IaaS, confs...)
		client.Request(i, "/checkFetch", "noauth:vmcheck", fmt.Sprintf("%s:10000", vmip))

		/// 50 ctns
		for k := 0; k < 50; k++ {

			ctnip := fmt.Sprintf("%d.%d.%d.%d", first, second, third, k)
			ctnid := fmt.Sprintf("vm%d-ctn%d", j, k)
			client.Request(i, "/postInstance", vmid,
				ctnid,
				"image-ctn",
				fmt.Sprintf("%s:1-65535", ctnip),
				"noauth:docker")
			confs2 := make([]string, 21)
			confs2[0] = ctnid
			for x := 1; x < 21; x++ {
				confs2[x] = fmt.Sprintf("cccc%d", x)
			}
			client.Request(i, "/postInstanceConfig", vmid, confs2...)
			client.Request(i, "/checkFetch", "noauth:ctncheck", fmt.Sprintf("%s:10000", ctnip))

			// 4 procs
			for l := 0; l < 4; l++ {
				port1 := 30000 + l*1000
				port2 := 30999 + l*1000
				pip := fmt.Sprintf("%d.%d.%d.%d:%d-%d", first, second, third, k,
					port1, port2)
				pid := fmt.Sprintf("vm%d-ctn%d-spark%d", j, k, l)
				client.Request(i, "/postInstance", ctnid,
					pid,
					"image-spark",
					pip,
					"noauth:noauth:spark")
				confs3 := make([]string, 21)
				confs3[0] = pid
				for x := 1; x < 21; x++ {
					confs3[x] = fmt.Sprintf("cccc%d", x)
				}
				client.Request(i, "/postInstanceConfig", ctnid, confs3...)
				client.Request(i, "/checkFetch", "noauth:proccheck", fmt.Sprintf("%s:%d", ctnip, port1+1))
				client.Request(i, "/lazyDeleteInstance", ctnid, pid)
			}
			client.Request(i, "/lazyDeleteInstance", vmid, ctnid)
		}

		client.Request(i, "/lazyDeleteInstance", IaaS, vmid)
	}

	done <- true
}
