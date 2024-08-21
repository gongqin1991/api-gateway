package main

func DumpRequest(request *DirectorRequest) {
	logger.WithContext(request.Context()).Infof("request:%s", request.RequestURI)
}
