class App:
    def __init__(self, appName, funcs, slo, adj):
        self.Nodes = []

        self.AppName = appName

        self.SLO = slo

        self.Adj = adj

        self.FuncNames = funcs

        self.ReadyFuncs = []

        self.FinishedFuncs = []

        self.generateNodes()

    def generateNodes(self):
        for idx, name in enumerate(self.FuncNames):
            node = Node(name, idx)
            self.Nodes.append(node)

        for i, connect in enumerate(self.Adj):
            for j, val in enumerate(connect):
                if val == 1:
                    # 父函数
                    fromN = self.Nodes[i]
                    # 子函数
                    toN = self.Nodes[j]
                    fromN.Children.append(toN)
                    toN.Parents.append(fromN)

        for node in self.Nodes:
            if len(node.Parent) == 0:
                node.Ready = True
                self.ReadyFuncs.append(node)


class Node:
    def __init__(self, idx, name):
        self.Idx = idx

        self.Name = name

        self.AvgLatency = 0

        self.AvgCPU = 0

        self.Children = []

        self.Parents = []

        self.FinishedTime = 0

        self.StartTime = 0

        self.Ready = False

        self.Finished = False

    def getDescendants(self):
        count = 1
        visited = [self.Idx]
        queue = [self]

        while len(queue) > 0:
            node = queue.pop(0)

            for c in node.Children:
                if c.Idx not in visited:
                    queue.append(c)

                    visited.append(c.Idx)
                    count += 1

        return count

    def updateStartTime(self):
        for pnode in self.Parents:
            # 寻找最大父亲节点结束时间
            if pnode.FinishedTime > pnode.StartTime:

                self.StartTime = pnode.FinishedTime


