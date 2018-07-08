
metadata    :name        => "app",
            :description => "Choria Management Backplane",
            :author      => "R.I.Pienaar <rip@devco.net>",
            :license     => "Apache-2.0",
            :version     => "0.0.1",
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

    output :health_feature,
           :description => "If the HealthCheckable interface is used"
           :display_as => "Health Feature"

    output :pause_feature,
           :description => "If the Pausableable interface is used"
           :display_as => "Circuit Breaker Feature"

    output :shutdown_feature,
           :description => "If the Stopable interface is used"
           :display_as => "Shutdown Feature"

    output :facts_feature,
           :description => "If the InfoSource interface is used"
           :display_as => "Facts Feature"

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
    action act, :description => "#{act.capitalize} the Circuit Breaker do
        display :always

        output :paused,
               :description => "Circuit Breaker pause state",
               :display_as => "Paused"

        summarize do
            aggregate summary(:paused)
        end
    end
end

