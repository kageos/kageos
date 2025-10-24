package app

type App struct {
}

type CreateAPPReq struct {
	Namespace string `json:"namespace"`
	AppName   string `json:"app_name"`
}

//func CreateAPP(ctx context.Context, req *CreateAPPReq) (*App, error) {
//
//}
