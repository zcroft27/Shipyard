# Copyright (C) 2007 Shijoe George <spanjikk@redhat.com>

# This file is part of the sos project: https://github.com/sosreport/sos
#
# This copyrighted material is made available to anyone wishing to use,
# modify, copy, or redistribute it subject to the terms and conditions of
# version 2 of the GNU General Public License.
#
# See the LICENSE file in the source distribution for further information.

from sos.report.plugins import Plugin, IndependentPlugin


class Nscd(Plugin, IndependentPlugin):

    short_desc = 'Name service caching daemon'
    plugin_name = 'nscd'
    profiles = ('services', 'identity', 'system')

    files = ('/etc/nscd.conf',)
    packages = ('nscd',)

    def setup(self):
        self.add_copy_spec("/etc/nscd.conf")

        options = self.file_grep(r"^\s*logfile", "/etc/nscd.conf")
        if len(options) > 0:
            for opt in options:
                fields = opt.split()
                self.add_copy_spec(fields[1])

# vim: set et ts=4 sw=4 :
