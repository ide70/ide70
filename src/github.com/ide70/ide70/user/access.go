package user

type Access struct {
	DefaultLevel string
	LoginUnit    string
	//UserDatabase    UserDatabase
	AutoLogin *AutoLogin
}

type AutoLogin struct {
	UserName string
	Password string
}
