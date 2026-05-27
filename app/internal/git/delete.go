package git

// DeleteSafe attempts to delete branch with git branch -d.
// Returns non-nil error if the branch is unmerged.
func DeleteSafe(name string) error {
	_, err := Run("branch", "-d", name)
	return err
}

// DeleteForce deletes branch with git branch -D, even if unmerged.
func DeleteForce(name string) error {
	_, err := Run("branch", "-D", name)
	return err
}
