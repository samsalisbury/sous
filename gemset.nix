{
  addressable = {
    dependencies = ["public_suffix"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0viqszpkggqi8hq87pqp0xykhvz60g99nwmkwsb0v45kc2liwxvk";
      type = "gem";
    };
    version = "2.5.2";
  };
  archive-tar-minitar = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1j666713r3cc3wb0042x0wcmq2v11vwwy5pcaayy5f0lnd26iqig";
      type = "gem";
    };
    version = "0.5.2";
  };
  arr-pm = {
    dependencies = ["cabin"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "07yx1g1nh4zdy38i2id1xyp42fvj4vl6i196jn7szvjfm0jx98hg";
      type = "gem";
    };
    version = "0.0.10";
  };
  backports = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "17pcz0z6jms5jydr1r95kf1bpk3ms618hgr26c62h34icy9i1dpm";
      type = "gem";
    };
    version = "3.8.0";
  };
  cabin = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0b3b8j3iqnagjfn1261b9ncaac9g44zrx1kcg81yg4z9i513kici";
      type = "gem";
    };
    version = "0.9.0";
  };
  childprocess = {
    dependencies = ["ffi"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "04cypmwyy4aj5p9b5dmpwiz5p1gzdpz6jaxb42fpckdbmkpvn6j1";
      type = "gem";
    };
    version = "0.7.1";
  };
  claide = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0az54rp691hc42yl1xyix2cxv58byhaaf4gxbpghvvq29l476rzc";
      type = "gem";
    };
    version = "1.0.2";
  };
  claide-plugins = {
    dependencies = ["cork" "nap" "open4"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0bhw5j985qs48v217gnzva31rw5qvkf7qj8mhp73pcks0sy7isn7";
      type = "gem";
    };
    version = "0.9.2";
  };
  clamp = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0jb6l4scp69xifhicb5sffdixqkw8wgkk9k2q57kh2y36x1px9az";
      type = "gem";
    };
    version = "1.0.1";
  };
  colored2 = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0jlbqa9q4mvrm73aw9mxh23ygzbjiqwisl32d8szfb5fxvbjng5i";
      type = "gem";
    };
    version = "3.1.2";
  };
  cork = {
    dependencies = ["colored2"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1g6l780z1nj4s3jr11ipwcj8pjbibvli82my396m3y32w98ar850";
      type = "gem";
    };
    version = "0.3.0";
  };
  danger = {
    dependencies = ["claide" "claide-plugins" "colored2" "cork" "faraday" "faraday-http-cache" "git" "kramdown" "no_proxy_fix" "octokit" "terminal-table"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "09yxbns8scncgmmx0lagbrr604m7i0cnj3g95h28x38azhy6s3bl";
      type = "gem";
    };
    version = "5.5.3";
  };
  danger-prose = {
    dependencies = ["danger"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "022bdc63cpk5kr9qf5l4981p69d9h9pz3q8h6rclgzbqwznsp35z";
      type = "gem";
    };
    version = "2.0.3";
  };
  dotenv = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1pgzlvs0sswnqlgfm9gkz2hlhkc0zd3vnlp2vglb1wbgnx37pjjv";
      type = "gem";
    };
    version = "2.2.1";
  };
  faraday = {
    dependencies = ["multipart-post"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1gyqsj7vlqynwvivf9485zwmcj04v1z7gq362z0b8zw2zf4ag0hw";
      type = "gem";
    };
    version = "0.13.1";
  };
  faraday-http-cache = {
    dependencies = ["faraday"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1hcp58ig28cbkzzv5is9dcyv1999981rhkdwd993l4l85ybiv2cm";
      type = "gem";
    };
    version = "1.3.1";
  };
  ffi = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "034f52xf7zcqgbvwbl20jwdyjwznvqnwpbaps9nk18v9lgb1dpx0";
      type = "gem";
    };
    version = "1.9.18";
  };
  ffi-aspell = {
    dependencies = ["ffi"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1djl9s71rkz8gvzs0wy2r0r3j8mx5gyb5l31jci58q878z6jj2h0";
      type = "gem";
    };
    version = "1.1.0";
  };
  fpm = {
    dependencies = ["archive-tar-minitar" "arr-pm" "backports" "cabin" "childprocess" "clamp" "ffi" "json" "pleaserun" "ruby-xz"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1fs56fil70j5xcpkin70sbw50616fr1869nwshqxk4sqvcf1v214";
      type = "gem";
    };
    version = "1.8.1";
  };
  git = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1waikaggw7a1d24nw0sh8fd419gbf7awh000qhsf411valycj6q3";
      type = "gem";
    };
    version = "1.3.0";
  };
  insist = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0bw3bdwns14mapbgb8cbjmr0amvwz8y72gyclq04xp43wpp5jrvg";
      type = "gem";
    };
    version = "1.0.0";
  };
  io-like = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "04nn0s2wmgxij3k760h3r8m1dgih5dmd9h4v1nn085yi824i5z6k";
      type = "gem";
    };
    version = "0.3.0";
  };
  json = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0qmj7fypgb9vag723w1a49qihxrcf5shzars106ynw2zk352gbv5";
      type = "gem";
    };
    version = "1.8.6";
  };
  kramdown = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "12k1dayq3dh20zlllfarw4nb6xf36vkd5pb41ddh0d0lndjaaf5f";
      type = "gem";
    };
    version = "1.15.0";
  };
  mdspell = {
    dependencies = ["ffi-aspell" "kramdown" "mixlib-cli" "mixlib-config" "rainbow"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1nh50m0id4mjfhvhdfliahl6w75pxy4jaawa56mav1qnd7ncnylm";
      type = "gem";
    };
    version = "0.2.0";
  };
  mixlib-cli = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0647msh7kp7lzyf6m72g6snpirvhimjm22qb8xgv9pdhbcrmcccp";
      type = "gem";
    };
    version = "1.7.0";
  };
  mixlib-config = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0s2ag6jz59r1gn8rbhf5c1g2mpbkc5jmz2fxh3n7hzv80dfzk42w";
      type = "gem";
    };
    version = "2.2.4";
  };
  multipart-post = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "09k0b3cybqilk1gwrwwain95rdypixb2q9w65gd44gfzsd84xi1x";
      type = "gem";
    };
    version = "2.0.0";
  };
  mustache = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1g5hplm0k06vwxwqzwn1mq5bd02yp0h3rym4zwzw26aqi7drcsl2";
      type = "gem";
    };
    version = "0.99.8";
  };
  nap = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0xm5xssxk5s03wjarpipfm39qmgxsalb46v1prsis14x1xk935ll";
      type = "gem";
    };
    version = "1.1.0";
  };
  no_proxy_fix = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "006dmdb640v1kq0sll3dnlwj1b0kpf3i1p27ygyffv8lpcqlr6sf";
      type = "gem";
    };
    version = "0.1.2";
  };
  octokit = {
    dependencies = ["sawyer"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0h6cm7bi0y7ysjgwws3paaipqdld6c0m0niazrjahhpz88qqq1g4";
      type = "gem";
    };
    version = "4.7.0";
  };
  open4 = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1cgls3f9dlrpil846q0w7h66vsc33jqn84nql4gcqkk221rh7px1";
      type = "gem";
    };
    version = "1.3.4";
  };
  pleaserun = {
    dependencies = ["cabin" "clamp" "dotenv" "insist" "mustache" "stud"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "17jm79fwa6rzxynsasd2xxgpdbpc11ig8as3iqhw4l1l7b0if7cw";
      type = "gem";
    };
    version = "0.0.29";
  };
  public_suffix = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0snaj1gxfib4ja1mvy3dzmi7am73i0mkqr0zkz045qv6509dhj5f";
      type = "gem";
    };
    version = "3.0.0";
  };
  rainbow = {
    dependencies = ["rake"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "08w2ghc5nv0kcq5b257h7dwjzjz1pqcavajfdx2xjyxqsvh2y34w";
      type = "gem";
    };
    version = "2.2.2";
  };
  rake = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "08acfrq4chxcnd03l1zwjdb7ginmx461db49p9hb7czy4ni2lhbx";
      type = "gem";
    };
    version = "12.2.1";
  };
  ruby-xz = {
    dependencies = ["ffi" "io-like"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "11bgpvvk0098ghvlxr4i713jmi2izychalgikwvdwmpb452r3ndw";
      type = "gem";
    };
    version = "0.2.3";
  };
  sawyer = {
    dependencies = ["addressable" "faraday"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "0sv1463r7bqzvx4drqdmd36m7rrv6sf1v3c6vswpnq3k6vdw2dvd";
      type = "gem";
    };
    version = "0.8.1";
  };
  stud = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1raavxgn5k4bxap5wqdl7zmfw5k4ndl8aagnajlfg4f0bmm8yni7";
      type = "gem";
    };
    version = "0.0.22";
  };
  terminal-table = {
    dependencies = ["unicode-display_width"];
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "1512cngw35hsmhvw4c05rscihc59mnj09m249sm9p3pik831ydqk";
      type = "gem";
    };
    version = "1.8.0";
  };
  unicode-display_width = {
    source = {
      remotes = ["https://rubygems.org"];
      sha256 = "12pi0gwqdnbx1lv5136v3vyr0img9wr0kxcn4wn54ipq4y41zxq8";
      type = "gem";
    };
    version = "1.3.0";
  };
}