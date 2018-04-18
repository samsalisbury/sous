# vim: ft=ruby:
#
def check_for_debug
  git.diff.reject do |file|
    file.path =~ /_test.go$/
  end.each do |file|
    file.patch.each_line do |patch_line|
      if /^\+[^+].*(spew\.Pr|spew\.Du|spew.\Fp|spew\.Fd)/ =~ patch_line
        fail "Debugging output: #{patch_line} (there may be others)"
        return
      end
    end
  end
end

def check_lgtm(image_url: nil, https_image_only: false)
  return unless status_report[:errors].length.zero? &&
    status_report[:warnings].length.zero?

  markdown(<<-lgtm.gsub(%r[^[ ]*(?=\S)]m, ""))
  # LGTM!

  No errors, no warnings. :shipit:
  lgtm
end

# BEGIN

# Sometimes it's a README fix, or something like that - which isn't relevant for
# including in a project's CHANGELOG for example
modified_app_files = git.modified_files.grep(/(?<!_test)\.go$/)

modified_app_files = modified_app_files.find_all do |file|
  diff = git.diff_for_file(file)
  interesting_lines = diff.patch.lines[5..-1]
  if !interesting_lines.nil?
    significant_lines = interesting_lines.grep_v(%r{^(?: |[+-]\s*(?://|$)|@@)})
    !significant_lines.empty?
  else
    false
  end
end
app_changed = !modified_app_files.empty?

declared_trivial = github.pr_title.include? "#trivial" || !app_changed

# Make it more obvious that a PR is a work in progress and shouldn't be merged yet
warn("PR is classed as Work in Progress") if github.pr_title.include? "WIP"

# Warn when there is a big PR
warn("Big PR: #{git.lines_of_code} lines of code changed") if git.lines_of_code > 500

if !git.modified_files.include?("CHANGELOG.md") && (app_changed && !declared_trivial)
  fail("Please include a CHANGELOG entry.", sticky: false)
end

full_prose = true
if full_prose
  prose.lint_files "docs/*.md"
else
  markdown_files = (modified_files + added_files).select do |line|
    line.start_with?("_posts") && (line.end_with?(".markdown") || line.end_with?(".md"))
  end

  # will check any .markdown files in this PR with proselint
  proselint.lint_files markdown_files
end

if !(prose.mdspell_installed? and prose.proselint_installed?)
  message "mdspell or proselint not available - prose not linted"
end

check_lgtm
check_for_debug
