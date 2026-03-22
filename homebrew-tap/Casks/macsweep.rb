cask "macsweep" do
  version "0.1.0"
  sha256 "5ac4342a29287cab4fbdd99a6057b1c6b7872a80f57a2600bd50789334c53884"

  url "https://github.com/Ashish1101/mac_sweep/releases/download/v#{version}/MacSweep-v#{version}-arm64.zip"
  name "MacSweep"
  desc "A safe, visual disk cleaner for macOS"
  homepage "https://github.com/Ashish1101/mac_sweep"

  app "macsweep.app"

  zap trash: [
    "~/.config/macsweep",
  ]
end
