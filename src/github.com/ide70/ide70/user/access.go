package user

type Access struct {
	DefaultLevel string
	LoginUnits   map[string]bool
	//UserDatabase    UserDatabase
	AutoLogin *AutoLogin
}

type AutoLogin struct {
	UserName string
	Password string
}
