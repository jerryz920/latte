package main

import (
	eurosys18 "github.com/jerryz920/conferences/eurosys18"
	jhttp "github.com/jerryz920/utils/goutils/http"
)

func main() {
	server := jhttp.NewAPIServer(nil)
	server.ListenAndServe(eurosys18.CmdProxyAddress)
}
