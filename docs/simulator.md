# Running the Simulator

WIP


## Disable OS X timer coalescing

OS X Mavericks introduced a power-saving feature called Timer Coalescing. Unfortunately, the feature also interferes with the network simulator timing. For any simulator (whether in-process or multi-process) to work correctly, you may need to turn off Timer Coalescing:

```sh
sudo sysctl -w kern.timer.coalescing_enabled=0
```

After you've finished running the simulator, you may restore the power-saving feature:

```sh
sudo sysctl -w kern.timer.coalescing_enabled=1
```

Thanks to [Tim Oliver][coalescing_disabled] for this command!


[coalescing_disabled]: https://gist.github.com/TimOliver/8612227
