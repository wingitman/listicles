# listicle shell integration for fish
# Source this file in your ~/.config/fish/config.fish:
#   source /path/to/listicle/shell/listicle.fish
#
# Or after `make install` it will be appended automatically.

function l --description "listicle: interactive directory navigator"
    set tmp (mktemp)
    listicle --cd-file $tmp $argv
    set dir (cat $tmp 2>/dev/null)
    rm -f $tmp
    if test -n "$dir" -a "$dir" != (pwd)
        builtin cd $dir
    end
end
