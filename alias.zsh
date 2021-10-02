alias scd() {
  target=$(go run ./cli/main.go fzf)
  cd $target
}
