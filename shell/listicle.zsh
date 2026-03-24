# listicle shell integration for zsh
# Source this file in your ~/.zshrc:
#   source /path/to/listicle/shell/listicle.zsh
#
# Or after `make install` it will be appended automatically.

l() {
    local tmp
    tmp=$(mktemp)
    listicle --cd-file "$tmp" "$@"
    local dir
    dir=$(cat "$tmp" 2>/dev/null)
    rm -f "$tmp"
    if [[ -n "$dir" && "$dir" != "$PWD" ]]; then
        builtin cd "$dir" || return 1
    fi
}
