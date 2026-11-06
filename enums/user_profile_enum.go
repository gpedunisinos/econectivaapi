package enums

type UserProfileEnum int16

const (
	UserProfileEnumUser          UserProfileEnum = 10
	UserProfileEnumAdministrator UserProfileEnum = 40
)

func (s UserProfileEnum) String() string {
	switch s {
	case UserProfileEnumUser:
		return "Usuário"
	case UserProfileEnumAdministrator:
		return "Administrador"
	}

	return "Desconhecido"
}

func (s UserProfileEnum) Value() int16 {
	return int16(s)
}
