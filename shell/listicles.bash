# listicles shell integration for bash
# Source this file in your ~/.bashrc:
#   source /path/to/listicles/shell/listicles.bash
#
# Or after `make install` it will be appended automatically.

l() {
    local tmp
    tmp=$(mktemp)
    listicles --cd-file "$tmp" "$@"
    local dir
    dir=$(cat "$tmp" 2>/dev/null)
    rm -f "$tmp"
    if [ -n "$dir" ] && [ "$dir" != "$PWD" ]; then
        builtin cd "$dir" || return 1
    fi
}
