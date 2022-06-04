import os
import random
import numpy as np
from GCNSELF import GCN
from GCN import GraphCNN
from Application import App

loads = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 20, 30, 40, 50]

'''
ReadFile
'''

class FileReader:
    def __init__(self):
        self.funcs = os.listdir("./data")

        self.latency = {}

        self.cpu = {}

        self.cluster = {}

        self.waste = {}

        for func in self.funcs:
            self.latency[func] = {}
            self.cpu[func] = {}
            self.cluster[func] = {}
            self.waste[func] = {}

            files = "./data/%s" % func
            for r, d, f in os.walk(files):
                if r == files:
                    continue
                cpu = int(r.split("/")[-1])
                self.latency[func][cpu] = {}
                self.cpu[func][cpu] = {}
                self.cluster[func][cpu] = {}
                self.waste[func][cpu] = {}

                for ff in f:
                    load = int(ff.split(".")[0])
                    self.latency[func][cpu][load] = [float(i.split(",")[0]) * 1000 for i in
                                                     open(r + "/" + ff, "r").readlines()]
                    self.cpu[func][cpu][load] = [float(i.split(",")[1]) for i in open(r + "/" + ff, "r").readlines()]

                    self.cluster[func][cpu][load] = [float(i.split(",")[2]) for i in open(r + "/" + ff, "r").readlines()]

                    self.waste[func][cpu][load] = [float(i.split(",")[3]) for i in open(r + "/" + ff, "r").readlines()]

    def query(self, Func, CPU, Load):
        return self.latency[Func][CPU][Load], \
               self.cpu[Func][CPU][Load]


    def avgLatency(self, Func):
        total_latency = 0
        length = 0

        for cpu in self.latency[Func]:
            total_latency += sum(self.latency[Func][cpu])
            for load in self.latency[Func][cpu]:
                length += len(self.latency[Func][cpu][load])
        return total_latency / length

    def avgCPU(self, Func):
        total_cpu = 0
        length = 0

        for cpu in self.cpu[Func]:
            total_cpu += sum(self.cpu[Func][cpu])
            for load in self.cpu[Func][cpu]:
                length += len(self.cpu[Func][cpu][load])
        return total_cpu / length

    def generateApp(self):
        r, d, _ = os.walk("data/applications")
        appName = r + "/" + d[random.randrange(0, len(d))]
        funcsFile = open(appName + "/func", "r").readlines()
        slo = funcsFile[0]
        funcs = []

        for f in funcsFile[1:]:
            funcs.append(f.replace("\n", ""))

        adj = np.zeros((len(funcs), len(funcs)))

        adjFile = open(appName + "/adj", "r").readlines()

        for line in adjFile:
            func = line.split(",")

            fromF = func[0]

            toF = func[1]

            adj[fromF][toF] = 1

        return App(appName, funcs, slo, adj)

    def generateClusterState(self, Func, load):

        return [load, self.cluster[Func]]

    def getWaste(self, Name, Cpu):
        return np.random.choice(self.waste[Name][Cpu][loads])


class ENV:
    def __init__(self):
        self.Reader = FileReader()

        self.observation_space = 9

        self.action_space = 100

        # The parameter of reward function
        self.mu = -0.02
        self.theta = -0.03
        self.alpha = -0.04

    def reset(self):
        self.load = loads[random.randint(0, len(loads) - 1)]

        self.APP = self.Reader.generateApp()

        self.avg_latency = {}

        self.avg_CPU = {}

        self.app_feature = []

        for func in self.APP.Nodes:
            avg_latency = self.Reader.avgLatency(func.Name)
            avg_cpu = self.Reader.avgCPU(func.Name)
            func.AvgLatency = avg_latency
            func.AvgCPU = avg_cpu
            self.app_feature.append([avg_latency, avg_cpu])

        self.stage = 0

        self.left_stage = len(self.APP.Adj)

        self.left_time = self.APP.SLO

        self.GCN = GCN(self.APP.Adj, self.app_feature)
        # self.GCN = GraphCNN()

    def observe(self):
        node = self.APP.ReadyFuncs.pop(0)

        node.updateStartTime()

        AppState = self.GCN.getFeature(node.Idx)

        RequestState = [node.Idx, self.APP.SLO - node.StartTime, node.getDescendants()]

        ClusterState = self.Reader.generateClusterState(node.Name, self.load)

        obs = np.concatenate((AppState, RequestState, ClusterState), axis=0)

        return obs, node

    def reward(self, CPU, Waste, node):

        if node.FinishedTime > self.APP.SLO:
            reward = self.mu * node.FinishedTime/self.APP.SLO * node.getDescendants()
        else:
            reward = self.theta * CPU + self.alpha * Waste

        return reward

    def step(self, action):
        Obs, node = self.observe()

        leastCPU = 1e9

        for key in self.Reader.cpu[node.Name].keys():
            leastCPU = min(leastCPU, key)

        CPU = action * 10 + leastCPU

        # The wasted of a function in a given CPU
        Waste = self.Reader.getWaste(node.Name, CPU)

        # The random latency of a function in a given CPU
        latency = self.Reader.cpu[node.Name][CPU][self.load]

        # The finished time with the start time
        node.FinishedTime = node.StartTime + latency

        # Turn the finished flag true
        node.Finished = True

        # Find the finished children node which have not been joint running queue
        for c in node.Children:
            ready = True
            for p in c.Parent:

                # Check if all of the parent have been finished
                if not p.Finished:
                    ready = False
                    break

            # Avoid repeative joining
            if ready and not c.Ready:
                c.Ready = True
                self.APP.ReadyFuncs.append(c)

        done = False

        # Determine whether there are unexecuted functions in the queue
        if len(self.APP.ReadyFuncs) == 0:
            done = True

        # Get the reward from reward function
        reward = self.reward(CPU, Waste, node)

        return Obs, reward, done, None
