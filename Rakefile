require 'rake'
begin
  require 'mcollective'
rescue LoadError
end

desc "Expands the action details section in a README.md file"
task :readme_expand do
  ddl_file = Dir.glob(File.join("agent/*.ddl")).first

  return unless ddl_file

  ddl = MCollective::DDL.new("package", :agent, false)
  ddl.instance_eval(File.read(ddl_file))

  lines = File.readlines("puppet/README.md").map do |line|
    if line =~ /^<\!--- actions -->/
      [
        "## Actions\n\n",
        "This agent provides the following actions, for details about each please run `mco plugin doc agent/%s`\n\n" % ddl.meta[:name]
      ] + ddl.entities.keys.sort.map do |action|
        " * **%s** - %s\n" % [action, ddl.entities[action][:description]]
      end
    else
      line
    end
  end.flatten

  File.open("puppet/README.md", "w") do |f|
    f.print lines.join
  end
end

desc "Set versions for a release"
task :prep_version do
  abort("Please specify VERSION") unless ENV["VERSION"]

  Rake::FileList["**/*.ddl"].each do |file|
    sh 'sed -i"" -re \'s/(\s+:version\s+=>\s+").+/\1%s",/\' %s' % [ENV["VERSION"], file]
  end
end

desc "Prepares for a release"
task :build_prep do
  if ENV["VERSION"]
    Rake::Task[:test].execute
    Rake::Task[:prep_version].execute
  end

  mkdir_p "puppet"

  cp "AGENT-README.md", "puppet/README.md"
  cp "CHANGELOG.md", "puppet"
  cp "LICENSE", "puppet"
  cp "NOTICE", "puppet"

  Rake::Task[:readme_expand].execute
end

desc "Builds the module found in the current directory, run build_prep first"
task :build do
  sh "/opt/puppetlabs/puppet/bin/mco plugin package --format aiomodulepackage --vendor choria"
end

desc "Release runs goreleaser to publish the binaries"
task :release do
  sh "goreleaser release"
end