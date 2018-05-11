package main

import (
	"fmt"

	base "github.com/jerryz920/conferences/latte/evaluation"
)

func Work(client *base.MetadataClient, i int, done chan bool) {
	block := NumVM * levels[NumLevel-1] / Nthread

	start := i * block
	end := (i + 1) * block

	for j := start; j < start+NumPerGroup+1; j++ {
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

		/// 50 ctns
		for k := 0; k < 10; k++ {

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

			// 4 procs
			for l := 0; l < 2 && NumLevel >= 3; l++ {
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
			}
		}
	}

	/// Creating the groups
	vm_master := fmt.Sprintf("vm%d", start)
	ctn_master := fmt.Sprintf("vm%d-ctn0", start)
	proc_master := fmt.Sprintf("vm%d-ctn0-spark0", start)
	client.Request(i, "/postCluster", vm_master, vm_master, "guard", "guard")
	client.Request(i, "/postCluster", ctn_master, ctn_master, "guard", "guard")
	client.Request(i, "/postCluster", proc_master, proc_master, "guard", "guard")

	for j, k := start+1, 0; j < end && k < NumPerGroup; j++ {
		vm_id := fmt.Sprintf("vm%d", j)
		ctn_id := fmt.Sprintf("vm%d-ctn0", j)
		proc_id := fmt.Sprintf("vm%d-ctn0-spark0", j)
		client.Request(i, "/postMembership", vm_master, vm_master, vm_id)
		client.Request(i, "/postAckMembership", vm_id, vm_master, vm_master)
		client.Request(i, "/postMembership", ctn_master, ctn_master, ctn_id)
		client.Request(i, "/postAckMembership", ctn_id, ctn_master, ctn_master)
		client.Request(i, "/postMembership", proc_master, proc_master, proc_id)
		client.Request(i, "/postAckMembership", proc_id, proc_master, proc_master)
		k++
	}
	client.Request(i, "/checkFetch", "noauth:vmcheck-master", vm_master)
	client.Request(i, "/checkFetch", "noauth:ctncheck-master", ctn_master)
	client.Request(i, "/checkFetch", "noauth:proccheck-master", proc_master)

	for j, k := start+1, 0; j < end && k < NumPerGroup; j++ {
		vm_id := fmt.Sprintf("vm%d", j)
		ctn_id := fmt.Sprintf("vm%d-ctn0", j)
		proc_id := fmt.Sprintf("vm%d-ctn0-spark0", j)

		client.Request(i, "/checkFetch", "noauth:vmcheck", vm_id)
		client.Request(i, "/checkFetch", "noauth:ctncheck", ctn_id)
		client.Request(i, "/checkFetch", "noauth:proccheck", proc_id)
		/// 50 ctns
		// 4 procs
		k++
	}

	done <- true
}
