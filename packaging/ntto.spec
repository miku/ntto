Summary:    Shrink N-Triples by applying namespace abbreviations.
Name:       ntto
Version:    0.4.2
Release:    0
License:    GPLv3
BuildArch:  x86_64
BuildRoot:  %{_tmppath}/%{name}-build
Group:      System/Base
Vendor:     UB Leipzig
URL:        https://github.com/miku/ntto

%description

ntto shrinks triples by applying namespace abbreviations.

%prep
# the set up macro unpacks the source bundle and changes in to the represented by
# %{name} which in this case would be my_maintenance_scripts. So your source bundle
# needs to have a top level directory inside called my_maintenance _scripts
# %setup -n %{name}

%build
# this section is empty for this example as we're not actually building anything

%install
# create directories where the files will be located
mkdir -p $RPM_BUILD_ROOT/usr/local/sbin

# put the files in to the relevant directories.
# the argument on -m is the permissions expressed as octal. (See chmod man page for details.)
install -m 755 ntto $RPM_BUILD_ROOT/usr/local/sbin

%post
# the post section is where you can run commands after the rpm is installed.
# insserv /etc/init.d/my_maintenance

%clean
rm -rf $RPM_BUILD_ROOT
rm -rf %{_tmppath}/%{name}
rm -rf %{_topdir}/BUILD/%{name}

# list files owned by the package here
%files
%defattr(-,root,root)
/usr/local/sbin/ntto


%changelog
* Mon Aug 4 2014 Martin Czygan
- 0.4.1 release
- updated RULES

* Thu Jul 31 2014 Martin Czygan
- 0.4.0 release
- now supports json output

* Tue Jul 29 2014 Martin Czygan
- 0.3.3 release
- use replace (mysql-server utility) or fallback to perl

* Wed Jul 23 2014 Martin Czygan
- 0.1 release
