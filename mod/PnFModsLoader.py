# PnFModsLoader.py
# Bootstrap loader for World of Warships PnFMods framework
import os
import sys

API_VERSION = "API_v1.0"

def init():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    pnf_mods_dir = os.path.join(current_dir, "PnFMods")

    if not os.path.isdir(pnf_mods_dir):
        return

    if current_dir not in sys.path:
        sys.path.insert(0, current_dir)

    for item in os.listdir(pnf_mods_dir):
        mod_dir = os.path.join(pnf_mods_dir, item)
        main_file = os.path.join(mod_dir, "Main.py")
        if os.path.isdir(mod_dir) and os.path.isfile(main_file):
            try:
                mod_name = "PnFMods." + item + ".Main"
                __import__(mod_name)
            except Exception as e:
                # Log or ignore errors in individual mods to avoid crashing client
                pass

try:
    init()
except Exception:
    pass
