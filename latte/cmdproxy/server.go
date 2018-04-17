package main

import (
	jhttp "github.com/jerryz920/utils/goutils/http"
	eurosys18 "github.com/jerryz920/utils/random/conferences/eurosys18"
)

func main() {
	server := jhttp.NewAPIServer(nil)
	server.ListenAndServe(eurosys18.CmdProxyAddress)
}
