
metadata    :name        => "backplane",
            :description => "Choria Management Backplane",
            :author      => "R.I.Pienaar <rip@devco.net>",
            :license     => "Apache-2.0",
            :version     => "1.1.0",
            :url         => "https://choria.io/",
            :timeout     => 10

action "info", :description => "Information about the managed service" do
    display :always

    output :backplane_version,
           :description => "Version of the Choria Backplane",
           :display_as => "Choria Backplane"

    output :version,
           :description => "Service Version",
           :display_as => "Version"

    output :paused,
           :description => "Circuit Breaker pause state",
           :display_as => "Paused"

    output :facts,
           :description => "Instance Facts",
           :display_as => "Facts"

    output :healthy,
           :description => "Health Status",
           :display_as => "Healthy"

    output :loglevel,
           :description => "Active log level",
           :display_as => "Log Level"

    output :healthcheck_feature,
           :description => "If the HealthCheckable interface is used",
           :display_as => "Health Feature"

    output :pause_feature,
           :description => "If the Pausableable interface is used",
           :display_as => "Circuit Breaker Feature"

    output :shutdown_feature,
           :description => "If the Stopable interface is used",
           :display_as => "Shutdown Feature"

    output :facts_feature,
           :description => "If the InfoSource interface is used",
           :display_as => "Facts Feature"

    output :loglevel_feature,
           :description => "If the LogLevelSetable interface is used",
           :display_as => "Log Level Feature"

    summarize do
        aggregate summary(:version)
        aggregate summary(:paused)
        aggregate summary(:healthy)
    end
end
    
action "ping", :description => "Backplane communications test" do
    output :version,
            :description => "The version of the Choria Backplane system in use",
            :display_as => "Choria Backplane"

    summarize do
        aggregate summary(:version)
    end   
end

action "health", :description => "Checks the health of the managed service" do
    output :result,
            :description => "The result from the check method",
            :display_as => "Result"

    output :healthy,
            :description => "Status indicator for the checked service",
            :display_as => "Healthy",
            :default => false

    summarize do
        aggregate summary(:healthy)
    end   
end

action "shutdown", :description => "Terminates the managed service" do
    output :delay,
            :description => "How long after running the action the shutdown will be initiated",
            :display_as => "Delay"
end

["pause", "resume", "flip"].each do |act|
    action act, :description => "#{act.capitalize} the Circuit Breaker" do
        display :always

        output :paused,
               :description => "Circuit Breaker pause state",
               :display_as => "Paused"

        summarize do
            aggregate summary(:paused)
        end
    end
end

["debuglvl", "infolvl", "warnlvl", "critlvl"].each do |act|
    action act, :description => "Set the logging level to #{act.gsub('lvl', '')}" do
        display :always

        output :level,
               :description => "Log level that was activated",
               :display_as => "Log Level"

        summarize do
            aggregate summary(:level)
        end
    end
end

