export const heroCode = `# Start the daemon
emeet-pixyd

# Check status
emeet-pixy status

# Auto mode handles everything
# Call starts  -> face tracking + noise cancellation + audio switch
# Call ends    -> privacy mode (hardware lens block)

# Manual control also available
emeet-pixy track          # Enable face tracking
emeet-pixy privacy        # Privacy mode
emeet-pixy center         # Center camera
emeet-pixy pan -90        # Pan left 90 degrees
emeet-pixy zoom 120       # Zoom to 120x`;
