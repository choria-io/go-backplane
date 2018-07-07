package main

var ddlTempl = `
metadata    :name        => "{{.Name}}",
            :description => "Choria Management Backplane",
            :author      => "R.I.Pienaar <rip@devco.net>",
            :license     => "Apache-2.0",
            :version     => "{{.Version}}",
            :url         => "https://choria.io/",
            :timeout     => 2

{{if .Pausable}}
["info", "pause", "resume", "flip"].each do |act|
    action act, :description => act do
        display :always

        output :paused,
               :description => "Circuit Breaker pause state",
               :display_as => "Paused"

        if act == "info"
            output :version,
                   :description => "Service Version",
                   :display_as => "Version"

            output :facts,
                   :description => "Instance Facts",
                   :display_as => "Facts"
        end

        summarize do
            aggregate summary(:version) if act == "info"
            aggregate summary(:paused)
        end
    end
end
{{end}}
`
