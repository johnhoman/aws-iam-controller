package controllers

type InvalidRoleStatusError string

func (err InvalidRoleStatusError) Error() string {
	return string(err)
}

func NewInvalidRoleStatus(message string) error {
	return InvalidRoleStatusError(message)
}

type ConflictError string

func (err ConflictError) Error() string {
	return string(err)
}

func NewConflict(message string) error {
	return ConflictError(message)
}
