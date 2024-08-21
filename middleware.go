package main

type MiddlewareFunc func(*DirectorRequest)

type DirectorMiddleware struct {
	Middleware MiddlewareFunc
}

func (p *DirectorMiddleware) Use(middlewares ...MiddlewareFunc) {
	chain := p.Middleware
	for i := range middlewares {
		middleware := middlewares[i]
		if middleware == nil {
			continue
		}
		old := chain
		chain = func(request *DirectorRequest) {
			if old != nil {
				old(request)
			}
			if request.state == Abort {
				return
			}
			middleware(request)
		}
	}
	p.Middleware = chain
}
