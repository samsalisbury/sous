# Sometimes it's a README fix, or something like that - which isn't relevant for
# including in a project's CHANGELOG for example
app_changes = !git.modified_files.grep(/(?<!_test)\.go$/).empty?
declared_trivial = github.pr_title.include? "#trivial" || !app_changes

# Make it more obvious that a PR is a work in progress and shouldn't be merged yet
warn("PR is classed as Work in Progress") if github.pr_title.include? "[WIP]"

# Warn when there is a big PR
warn("Big PR: #{git.lines_of_code} lines of code changed") if git.lines_of_code > 500

if !git.modified_files.include?("CHANGELOG.md") && (app_changes && !declared_trivial)
  fail("Please include a CHANGELOG entry.", sticky: false)
end

lgtm.check_lgtm

p git.diff.full_diff
